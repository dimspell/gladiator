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

var _ Redirect = (*ListenerTCP)(nil)

type ListenerTCP struct {
	mu     sync.RWMutex
	logger *slog.Logger

	listener   TCPListener
	conn       TCPConn
	closed     bool
	lastActive time.Time
}

type TCPListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}

type TCPConn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	SetReadDeadline(t time.Time) error
	RemoteAddr() net.Addr
}

// ListenTCP initializes a TCP listener on the given IP and port.
func ListenTCP(ipv4 string, portNumber string) (*ListenerTCP, error) {
	if net.ParseIP(ipv4) == nil {
		return nil, fmt.Errorf("listen-tcp: invalid IPv4 address format")
	}

	if portNumber == "" {
		portNumber = defaultTcpPort
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, err
	}

	logger := slog.With(
		slog.String("redirect", "listen-tcp"),
		slog.String("address", listener.Addr().String()),
	)
	logger.Info("TCP listener started")

	return &ListenerTCP{
		listener: listener,
		logger:   logger,
	}, nil
}

// Run listens for incoming TCP connection from the game client and forwards the
// received data.
func (p *ListenerTCP) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	go func() {
		<-ctx.Done()
		p.logger.Info("Listener shutting down due to context cancellation")
		_ = p.Close()
	}()

	conn, err := p.listener.Accept()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("failed to accept TCP connection: %w", err)
	}

	p.logger.Debug("Accepted new connection", "remote-addr", conn.RemoteAddr())

	// Store the first active connection
	p.mu.Lock()
	p.conn = conn
	p.lastActive = time.Now()
	p.mu.Unlock()

	if err := p.handleConnection(ctx, conn, onReceive); err != nil {
		p.logger.Error("Failed to handle connection", "remote-addr", conn.RemoteAddr(), "error", err)
		return err
	}
	return nil
}

// handleConnection reads from the TCP connection and forwards the data received
// from the game client.
func (p *ListenerTCP) handleConnection(ctx context.Context, conn TCPConn, onReceive func(p []byte) (err error)) error {
	defer func() {
		if err := conn.Close(); err != nil {
			p.logger.Warn("Error closing TCP connection", logging.Error(err))
		}
	}()

	// Handle incoming data from the game client
	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buf)

			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

			n, err := conn.Read(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					p.lastActive = time.Now()
					continue
				}
				if errors.Is(err, io.EOF) {
					return fmt.Errorf("game client has closed the TCP connection: %w", err)
				}
				if errors.Is(err, net.ErrClosed) {
					return fmt.Errorf("listener has closed the connection: %w", err)
				}
				if errors.Is(err, io.ErrClosedPipe) {
					return nil
				}

				return fmt.Errorf("failed to read data: %w", err)
			}

			p.lastActive = time.Now()
			p.logger.Debug("Received packet from the game client", "size", n, "data", buf[:n])

			if err := onReceive(buf[:n]); err != nil {
				p.logger.Warn("Failed to write data", logging.Error(err))
				return fmt.Errorf("failed to write to data channel: %w", err)
			}
		}
	}
}

// Write sends data to the active TCP connection (game client).
func (p *ListenerTCP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return 0, fmt.Errorf("listen-tcp: no active connection")
	}

	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send data", logging.Error(err))
		return n, fmt.Errorf("listen-tcp: write failed: %w", err)
	}

	p.logger.Debug("Sent to the game client", "size", n, "data", msg[:n])
	return n, nil
}

// Close shuts down the listener and any active connection.
func (p *ListenerTCP) Close() error {
	p.logger.Info("Closing TCP listener")

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("listen-tcp: already closed")
	}

	// Close active TCP connection if present
	var err error
	if p.conn != nil {
		err = p.conn.Close()
	}

	// Close the TCP listener
	err = errors.Join(err, p.listener.Close())
	p.closed = true
	return err
}

const defaultTimeout = time.Second * 5

func (p *ListenerTCP) Alive(now time.Time) bool {
	p.mu.RLock()
	alive := !p.closed && p.conn != nil && p.lastActive.After(now.Add(-defaultTimeout))
	p.mu.RUnlock()
	return alive
}

func StartProbeTCP(ctx context.Context, addr string, onDisconnect func()) error {
	logger := slog.With("component", "probe-tcp")

	// Check if the connection to the game server can be established
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		return fmt.Errorf("could not connect to game server: %w", err)
	}

	// Check if the game server is still running
	go func() {
		defer func() {
			onDisconnect()
			_ = conn.Close()
		}()

		time.Sleep(10 * time.Second)

		buf := make([]byte, 1)
		for {
			select {
			case <-ctx.Done():
				logger.Info("Context cancelled")
				return
			default:
				_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

				if _, err := conn.Read(buf); err != nil {
					var ne net.Error
					if errors.As(err, &ne) && ne.Timeout() {
						continue
					}
					if errors.Is(err, io.EOF) {
						logger.Debug("[TCP Probe] listener host has closed the connection")
						return
					}
					if errors.Is(err, net.ErrClosed) {
						logger.Debug("[TCP Probe] probe has closed the connection")
						return
					}
					logger.Info("Connection to the listener is closed", logging.Error(err))
					return
				}
				continue
			}
		}
	}()

	return nil
}
