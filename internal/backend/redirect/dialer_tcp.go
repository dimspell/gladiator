package redirect

import (
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

// Ensure DialerTCP implements Redirect interface
var _ Redirect = (*DialerTCP)(nil)

type DialerTCP struct {
	mu         sync.RWMutex
	conn       TCPConn
	OnReceive  ReceiveFunc
	logger     *slog.Logger
	lastActive time.Time
}

// NewDialTCP establishes a TCP connection with the given IPv4 and port.
func NewDialTCP(ipv4 string, portNumber string, onReceive ReceiveFunc) (*DialerTCP, error) {
	if portNumber == "" {
		portNumber = defaultTCPPort
	}
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(ipv4, portNumber), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to game server on %s: %w", portNumber, err)
	}

	logger := slog.With(
		slog.String("redirect", "dial-tcp"),
		slog.String("local", tcpConn.LocalAddr().String()),
		slog.String("remote", tcpConn.RemoteAddr().String()),
	)
	logger.Info("Successfully connected via TCP")

	return &DialerTCP{
		conn:       tcpConn,
		OnReceive:  onReceive,
		logger:     logger,
		lastActive: time.Now(),
	}, nil
}

// Run handles reading from TCP and forwards data received from the game client.
func (p *DialerTCP) Run(ctx context.Context) error {
	defer func() {
		if err := p.Close(); err != nil {
			p.logger.Error("Error during TCP connection close", logging.Error(err))
		}
	}()

	buf := make([]byte, 1024)

	for {
		if p.conn == nil {
			return fmt.Errorf("tcp-dial: tcp connection is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buf)
			p.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := p.conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					p.logger.Info("Connection closed by server")
					return err
				}
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue
				}

				p.logger.Error("TCP read error", logging.Error(err))
				return err
			}

			p.lastActive = time.Now()

			if err := p.OnReceive(buf[:n]); err != nil {
				return fmt.Errorf("tcp-dial: failed to handle data received from the game client to: %w", err)
			}
		}
	}
}

// Write sends a message over the TCP connection to the game client.
func (p *DialerTCP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return 0, fmt.Errorf("tcp-dial: tcp connection is nil")
	}
	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send message", logging.Error(err))
		return n, err
	}
	p.lastActive = time.Now()
	return n, nil
}

// Close terminates the TCP connection.
func (p *DialerTCP) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return nil // Already closed or never opened
	}
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close TCP connection", logging.Error(err))
		return err
	}
	p.conn = nil // Prevent double close
	p.logger.Info("TCP connection closed")
	return nil
}

// Alive reports whether the TCP dialer is alive based on the last activity time and a timeout.
func (p *DialerTCP) Alive(now time.Time, timeout time.Duration) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return false
	}
	return p.lastActive.After(now.Add(-timeout))
}
