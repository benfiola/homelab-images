package internal

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
	"golang.org/x/net/ipv4"
)

const (
	packetBuffer = 512
	dedupTTL     = 500 * time.Millisecond
)

var (
	mdnsAddr = net.UDPAddr{
		IP:   net.ParseIP("224.0.0.251"),
		Port: 5353,
	}
)

type dedupCache struct {
	mu      sync.Mutex
	entries map[[sha256.Size]byte]time.Time
	ttl     time.Duration
}

func newDedupCache() *dedupCache {
	return &dedupCache{
		entries: make(map[[sha256.Size]byte]time.Time),
		ttl:     dedupTTL,
	}
}

func (d *dedupCache) seen(pkt []byte) bool {
	h := sha256.Sum256(pkt)
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for k, t := range d.entries {
		if now.Sub(t) > d.ttl {
			delete(d.entries, k)
		}
	}
	if _, ok := d.entries[h]; ok {
		return true
	}
	d.entries[h] = now
	return false
}

func (r *MDNSReflector) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	logger.Info("starting mdns reflector", "interfaces", r.Interfaces)

	conns := make([]*net.UDPConn, 0, len(r.Interfaces))
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	for _, ifName := range r.Interfaces {
		iface, err := net.InterfaceByName(ifName)
		if err != nil {
			return fmt.Errorf("failed to get interface %s: %w", ifName, err)
		}

		conn, err := net.ListenMulticastUDP("udp4", iface, &mdnsAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on multicast address for interface %s: %w", ifName, err)
		}

		p := ipv4.NewPacketConn(conn)
		if err := p.SetMulticastTTL(255); err != nil {
			return fmt.Errorf("failed to set multicast TTL on interface %s: %w", ifName, err)
		}
		if err := p.SetMulticastInterface(iface); err != nil {
			return fmt.Errorf("failed to set multicast interface on %s: %w", ifName, err)
		}

		conns = append(conns, conn)
		logger.Debug("listening on interface", "interface", ifName)
	}

	logger.Info("mDNS reflector started", "interfaces", len(conns))

	var wg sync.WaitGroup
	errChan := make(chan error, len(conns))

	for i, conn := range conns {
		wg.Add(1)
		ifName := r.Interfaces[i]
		go func(idx int, srcConn *net.UDPConn, srcName string) {
			defer wg.Done()
			cache := newDedupCache()
			if err := r.reflectPackets(ctx, idx, srcConn, srcName, conns, r.Interfaces, cache); err != nil {
				errChan <- fmt.Errorf("reflection failed on %s: %w", srcName, err)
			}
		}(i, conn, ifName)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

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

func (r *MDNSReflector) reflectPackets(ctx context.Context, idx int, srcConn *net.UDPConn, srcIfName string, allConns []*net.UDPConn, allIfNames []string, cache *dedupCache) error {
	logger := logging.FromContext(ctx)
	logger.Debug("reflectPackets started", "source_interface", srcIfName)

	buf := make([]byte, packetBuffer)
	consecutiveErrors := 0
	const maxConsecutiveErrors = 10

	backoffDuration := 100 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		srcConn.SetReadDeadline(time.Now().Add(1 * time.Second))

		n, _, err := srcConn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Debug("read timeout on interface (no packets received)", "interface", srcIfName)
				consecutiveErrors = 0
				continue
			}
			if ctx.Err() != nil {
				return nil
			}

			consecutiveErrors++
			logger.Warn("read error on interface", "interface", srcIfName, "error", err, "consecutive_errors", consecutiveErrors, "backoff_ms", backoffDuration.Milliseconds())

			if consecutiveErrors >= maxConsecutiveErrors {
				return fmt.Errorf("too many consecutive read errors on %s: %w", srcIfName, err)
			}

			select {
			case <-time.After(backoffDuration):
				backoffDuration = min(backoffDuration*2, maxBackoff)
			case <-ctx.Done():
				return nil
			}
			continue
		}

		consecutiveErrors = 0
		backoffDuration = 100 * time.Millisecond

		packet := buf[:n]
		logger.Debug("received mDNS packet", "source_interface", srcIfName, "packet_size", n)

		if cache.seen(packet) {
			logger.Debug("dropping duplicate mDNS packet", "source_interface", srcIfName, "packet_size", n)
			continue
		}

		for i, destConn := range allConns {
			if i == idx {
				continue
			}
			destIfName := allIfNames[i]
			_, err := destConn.WriteToUDP(packet, &mdnsAddr)
			if err != nil {
				logger.Error("failed to write to interface", "source", srcIfName, "dest", destIfName, "error", err)
			} else {
				logger.Debug("forwarded mDNS packet", "source", srcIfName, "dest", destIfName, "packet_size", n)
			}
		}
	}
}
