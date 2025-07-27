package redirect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
)

// Ensure ListenerUDP implements Redirect interface
var _ Redirect = (*ListenerUDP)(nil)

// ListenerUDP implements a UDP listener that can receive and forward UDP packets from a game client.
// It implements the Redirect interface.
type ListenerUDP struct {
	sync.Mutex
	logger     *slog.Logger
	conn       UDPConn
	lastActive time.Time
	OnReceive  ReceiveFunc
	remoteAddr *net.UDPAddr
}

// NewListenerUDP initializes the UDP listener on the given IP and port.
// It returns a ListenerUDP instance or an error if the listener cannot be started.
func NewListenerUDP(ipv4 string, portNumber string, onReceive ReceiveFunc) (*ListenerUDP, error) {
	if portNumber == "" {
		portNumber = defaultUDPPort
	}
	listenerAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("listen-udp: failed to resolve address: %w", err)
	}
	listenerConn, err := net.ListenUDP("udp", listenerAddr)
	if err != nil {
		return nil, fmt.Errorf("listen-udp: failed to listen on UDP: %w", err)
	}

	logger := slog.With(
		slog.String("redirect", "listen-udp"),
		slog.String("address", listenerAddr.String()),
	)
	logger.Info("UDP listener started")

	p := ListenerUDP{
		conn:      listenerConn,
		OnReceive: onReceive,
		logger:    logger,
	}
	return &p, nil
}

// Run starts the UDP listener loop, handling handshakes and forwarding packets.
// It blocks until the context is cancelled or an error occurs.
func (p *ListenerUDP) Run(ctx context.Context) error {
	defer p.Close()

	for {
		if p.conn == nil {
			return fmt.Errorf("conn is nil")
		}
		if err := p.handleHandshake(p.conn, p.OnReceive); err != nil {
			p.logger.Warn("Failed to handle handshake", logging.Error(err))
			continue
		}

		p.logger.Debug("Successful handshake")
		break
	}

	if err := p.handleConnection(ctx, p.conn, p.OnReceive); err != nil {
		p.logger.Error("Failed to handle connection", "error", err)
		return err
	}
	return nil
}

// handleHandshake waits for the initial handshake packet from a client and records the remote address.
// Returns an error if the handshake fails or a client is already connected.
func (p *ListenerUDP) handleHandshake(conn UDPConn, onReceive ReceiveFunc) error {
	p.Lock()
	defer p.Unlock()

	if p.remoteAddr != nil {
		return fmt.Errorf("someone is already connected")
	}

	buf := make([]byte, 4)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, remoteAddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return err
	}

	if !bytes.Equal(buf[:n], []byte{26, 0, 2, 0}) {
		return fmt.Errorf("invalid first packet, got: %v", buf[:n])
	}

	if err := onReceive(buf[:n]); err != nil {
		return fmt.Errorf("failed to forward data: %w", err)
	}

	p.remoteAddr = remoteAddr
	p.lastActive = time.Now()
	return nil
}

// handleConnection processes incoming UDP packets from the connected client.
// It calls the provided onReceive callback for each valid packet.
func (p *ListenerUDP) handleConnection(ctx context.Context, conn UDPConn, onReceive ReceiveFunc) error {
	buf := make([]byte, 1024)

	for {
		if conn == nil {
			return fmt.Errorf("listen-udp: connection is closed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buf)
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					p.lastActive = time.Now()
					continue
				}
				if errors.Is(err, io.EOF) {
					return fmt.Errorf("game client has closed the UDP connection: %w", err)
				}
				if errors.Is(err, net.ErrClosed) {
					return fmt.Errorf("udp-listener has closed the connection: %w", err)
				}

				p.logger.Warn("Failed to read UDP message", logging.Error(err))
				return fmt.Errorf("listen-udp: read error: %w", err)
			}

			// Ignore packets from other sources
			if !remoteAddr.IP.Equal(p.remoteAddr.IP) || remoteAddr.Port != p.remoteAddr.Port {
				p.logger.Warn("Received packet from an unknown source", "data", buf[:n], "remoteAddr", remoteAddr, "length", n)
				//continue
			}

			p.lastActive = time.Now()

			// Forward the packet to the game server
			if err := onReceive(buf[:n]); err != nil {
				p.logger.Warn("Failed to write message", logging.Error(err), "payload", buf[:n])
				return fmt.Errorf("listen-udp: write error: %w", err)
			}
		}
	}
}

// Write sends data to the last received remote address (the game client).
// Returns the number of bytes written or an error if the connection is closed or unavailable.
func (p *ListenerUDP) Write(msg []byte) (int, error) {
	p.Lock()
	defer p.Unlock()
	if p.remoteAddr == nil || p.conn == nil {
		return 0, fmt.Errorf("listen-udp: no remote address set or closed")
	}
	n, err := p.conn.WriteTo(msg, p.remoteAddr)
	if err != nil {
		p.logger.Warn("Failed to send UDP message", logging.Error(err))
		return n, fmt.Errorf("listen-udp: send failed: %w", err)
	}
	p.lastActive = time.Now()
	return n, nil
}

// Close immediately closes all active UDP connections and releases resources.
// It is safe to call multiple times.
func (p *ListenerUDP) Close() error {
	p.Lock()
	defer p.Unlock()

	if p.conn == nil {
		// Idempotent: do not error if already closed
		return nil
	}

	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}

	p.logger.Info("UDP listener closed")
	return nil
}

// Alive reports whether the UDP listener is alive based on the last activity time and a timeout.
func (p *ListenerUDP) Alive(now time.Time, timeout time.Duration) bool {
	p.Lock()
	defer p.Unlock()
	if p.conn == nil {
		return false
	}
	if p.remoteAddr == nil {
		return false
	}
	return p.lastActive.After(now.Add(-timeout))
}
