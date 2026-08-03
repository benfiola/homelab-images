package internal

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

const relayClientIdleTimeout = 5 * time.Minute

// UDPRelay sits in front of the game's UDP port for the container's entire
// life and forwards to the game on an internal-only port. This is what
// makes pause mode possible without raw packet capture: the kernel keeps
// delivering packets to a socket we own regardless of whether the backend
// is running or SIGSTOPped, so we can notice a reconnect attempt and wake
// the game ourselves.
type UDPRelay struct {
	publicAddr  string
	backendAddr *net.UDPAddr
	onPacket    func() // called before every forwarded packet; wakes the game if paused

	mu      sync.Mutex
	clients map[string]*relayClient
}

type relayClient struct {
	conn       *net.UDPConn
	lastActive time.Time
}

func NewUDPRelay(publicAddr, backendAddr string, onPacket func()) (*UDPRelay, error) {
	backend, err := net.ResolveUDPAddr("udp", backendAddr)
	if err != nil {
		return nil, err
	}
	return &UDPRelay{
		publicAddr:  publicAddr,
		backendAddr: backend,
		onPacket:    onPacket,
		clients:     make(map[string]*relayClient),
	}, nil
}

// Run binds the public port and forwards packets until ctx is cancelled.
func (r *UDPRelay) Run(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	publicAddr, err := net.ResolveUDPAddr("udp", r.publicAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", publicAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	logger.Info("udp relay listening", "address", r.publicAddr, "backend", r.backendAddr.String())

	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go r.evictIdleClients(ctx)

	buf := make([]byte, 65536)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logger.Warn("udp relay read error", "error", err)
			continue
		}

		r.onPacket()

		client, err := r.clientFor(ctx, conn, clientAddr)
		if err != nil {
			logger.Warn("udp relay: failed to reach backend", "client", clientAddr.String(), "error", err)
			continue
		}
		if _, err := client.conn.Write(buf[:n]); err != nil {
			logger.Warn("udp relay: forward to backend failed", "client", clientAddr.String(), "error", err)
		}
	}
}

// clientFor returns the backend connection for clientAddr, dialing a new
// one (and starting its reply pump) the first time this client is seen.
func (r *UDPRelay) clientFor(ctx context.Context, publicConn *net.UDPConn, clientAddr *net.UDPAddr) (*relayClient, error) {
	key := clientAddr.String()

	r.mu.Lock()
	if client, ok := r.clients[key]; ok {
		client.lastActive = time.Now()
		r.mu.Unlock()
		return client, nil
	}
	r.mu.Unlock()

	backendConn, err := net.DialUDP("udp", nil, r.backendAddr)
	if err != nil {
		return nil, err
	}
	client := &relayClient{conn: backendConn, lastActive: time.Now()}

	r.mu.Lock()
	r.clients[key] = client
	r.mu.Unlock()

	go r.pumpReplies(ctx, publicConn, clientAddr, client)
	return client, nil
}

// pumpReplies relays backend->client traffic for one client. Exits once its
// backend connection is closed (idle eviction) or errors out.
func (r *UDPRelay) pumpReplies(ctx context.Context, publicConn *net.UDPConn, clientAddr *net.UDPAddr, client *relayClient) {
	logger := logging.FromContext(ctx)
	buf := make([]byte, 65536)
	for {
		n, err := client.conn.Read(buf)
		if err != nil {
			return
		}
		if _, err := publicConn.WriteToUDP(buf[:n], clientAddr); err != nil {
			logger.Warn("udp relay: forward to client failed", "client", clientAddr.String(), "error", err)
			return
		}
	}
}

func (r *UDPRelay) evictIdleClients(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()
			for key, client := range r.clients {
				if time.Since(client.lastActive) > relayClientIdleTimeout {
					client.conn.Close()
					delete(r.clients, key)
				}
			}
			r.mu.Unlock()
		}
	}
}
