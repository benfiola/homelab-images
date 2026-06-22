package internal

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

const (
	packetBuffer = 512
)

var (
	mdnsAddr = net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}
)

func (r *MDNSReflector) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Info("starting mdns reflector", "source_interfaces", r.SourceInterfaces, "dest_interfaces", r.DestInterfaces)

	// Create listening connections for each source interface
	srcConns := make([]*net.UDPConn, 0, len(r.SourceInterfaces))
	srcIfNames := r.SourceInterfaces
	defer func() {
		for _, conn := range srcConns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	for _, ifName := range r.SourceInterfaces {
		iface, err := net.InterfaceByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to get source interface %s: %w", ifName, err)
		}

		conn, err := net.ListenMulticastUDP("udp4", iface, &mdnsAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on multicast address for interface %s: %w", ifName, err)
		}

		srcConns = append(srcConns, conn)
		logger.Debug("listening on interface", "interface", ifName)
	}

	// Create outgoing connections for each destination interface
	destConns := make([]*net.UDPConn, 0, len(r.DestInterfaces))
	destIfNames := r.DestInterfaces
	defer func() {
		for _, conn := range destConns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	for _, ifName := range r.DestInterfaces {
		iface, err := net.InterfaceByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to get destination interface %s: %w", ifName, err)
		}

		conn, err := net.ListenMulticastUDP("udp4", iface, &mdnsAddr)
		if err != nil {
			return fmt.Errorf("failed to create outgoing connection for interface %s: %w", ifName, err)
		}

		destConns = append(destConns, conn)
		logger.Debug("created outgoing connection on interface", "interface", ifName)
	}

	logger.Info("mDNS reflector started", "listening_on", len(srcConns), "forwarding_to", len(destConns))

	// Set up packet reflection
	var wg sync.WaitGroup
	errChan := make(chan error, len(srcConns))

	for i, conn := range srcConns {
		wg.Add(1)
		srcIfName := srcIfNames[i]
		go func(srcConn *net.UDPConn, srcName string) {
			defer wg.Done()
			if err := r.reflectPackets(ctx, srcConn, srcName, destConns, destIfNames); err != nil {
				errChan <- fmt.Errorf("reflection failed on %s: %w", srcName, err)
			}
		}(conn, srcIfName)
	}

	// Wait for either context cancellation or a goroutine error
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Block until error or context done
	select {
	case <-ctx.Done():
		logger.Info("shutting down mDNS reflector")
	case err := <-errChan:
		if err != nil {
			logger.Error("reflector encountered error", "error", err)
		}
	}

	wg.Wait()
	return nil
}

func (r *MDNSReflector) reflectPackets(ctx context.Context, srcConn *net.UDPConn, srcIfName string, destConns []*net.UDPConn, destIfNames []string) error {
	logger := logging.FromContext(ctx)
	logger.Debug("reflectPackets started", "source_interface", srcIfName)

	buf := make([]byte, packetBuffer)
	consecutiveErrors := 0
	const maxConsecutiveErrors = 10

	backoffDuration := 100 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		// Set read deadline to allow context checks
		srcConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, _, err := srcConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Debug("read timeout on source interface (no packets received)", "interface", srcIfName)
				consecutiveErrors = 0
				continue
			}
			if ctx.Err() != nil {
				return nil
			}

			consecutiveErrors++
			logger.Warn("read error on source interface", "interface", srcIfName, "error", err, "consecutive_errors", consecutiveErrors, "backoff_ms", backoffDuration.Milliseconds())

			// Give up after too many consecutive errors
			if consecutiveErrors >= maxConsecutiveErrors {
				return fmt.Errorf("too many consecutive read errors on %s: %w", srcIfName, err)
			}

			// Back off before retrying
			select {
			case <-time.After(backoffDuration):
				// Increase backoff exponentially
				backoffDuration = min(backoffDuration*2, maxBackoff)
			case <-ctx.Done():
				return nil
			}
			continue
		}

		// Reset on successful read
		consecutiveErrors = 0
		backoffDuration = 100 * time.Millisecond

		packet := buf[:n]
		logger.Debug("received mDNS packet", "source_interface", srcIfName, "packet_size", n)

		// Forward to all destination interfaces
		for i, destConn := range destConns {
			destIfName := destIfNames[i]
			_, err := destConn.WriteToUDP(packet, &mdnsAddr)
			if err != nil {
				logger.Error("failed to write to destination interface", "source", srcIfName, "dest", destIfName, "error", err)
			} else {
				logger.Debug("forwarded mDNS packet", "source", srcIfName, "dest", destIfName, "packet_size", n)
			}
		}
	}
}
