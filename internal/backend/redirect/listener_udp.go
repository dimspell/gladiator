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

	p.Lock()
	conn := p.conn
	p.Unlock()
	if conn == nil {
		return fmt.Errorf("conn is nil")
	}

	for {
		if err := p.handleHandshake(conn, p.OnReceive); err != nil {
			p.logger.Warn("Failed to handle handshake", logging.Error(err))
			continue
		}

		p.logger.Debug("Successful handshake")
		break
	}

	// The peer address is immutable after the handshake, so snapshot it once
	// and use the local copy in the connection loop. This avoids locking on the
	// hot path and any race with Write/Close.
	p.Lock()
	peerAddr := p.remoteAddr
	p.Unlock()

	if err := p.handleConnection(ctx, conn, peerAddr, p.OnReceive); err != nil {
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

	buf := make([]byte, 64)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read until we have at least the 4-byte magic, or the read fails.
	var n int
	var remoteAddr *net.UDPAddr
	for {
		readN, addr, err := conn.ReadFromUDP(buf[n:])
		if err != nil {
			return err
		}
		if remoteAddr == nil {
			remoteAddr = addr
		}
		n += readN
		if n >= 4 {
			break
		}
	}

	if !bytes.Equal(buf[:4], []byte{26, 0, 2, 0}) {
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
func (p *ListenerUDP) handleConnection(ctx context.Context, conn UDPConn, peerAddr *net.UDPAddr, onReceive ReceiveFunc) error {
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
					p.setLastActive()
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

			// Drop packets from a source other than the handshake-recorded peer.
			// This prevents local processes from spoofing packets into the game stream.
			if peerAddr == nil || !remoteAddr.IP.Equal(peerAddr.IP) || remoteAddr.Port != peerAddr.Port {
				p.logger.Warn("Received packet from an unknown source", "data", buf[:n], "remoteAddr", remoteAddr, "length", n)
				continue
			}

			p.setLastActive()

			// Forward the packet to the game server
			if err := onReceive(buf[:n]); err != nil {
				p.logger.Warn("Failed to write message", logging.Error(err), "payload", buf[:n])
				return fmt.Errorf("listen-udp: write error: %w", err)
			}
		}
	}
}

// setLastActive records the last activity time under the mutex so concurrent
// writers (handleConnection, Write) and readers (Alive) cannot race.
func (p *ListenerUDP) setLastActive() {
	p.Lock()
	p.lastActive = time.Now()
	p.Unlock()
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

	err := p.conn.Close()
	p.conn = nil
	p.logger.Info("UDP listener closed")
	return err
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
