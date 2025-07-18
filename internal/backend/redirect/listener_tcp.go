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
}

// ListenTCP initializes a TCP listener on the given IP and port.
func ListenTCP(ipv4 string, portNumber string) (*ListenerTCP, error) {
	if net.ParseIP(ipv4) == nil {
		return nil, fmt.Errorf("listen-tcp: invalid IPv4 address format")
	}

	if portNumber == "" {
		portNumber = defaultTCPPort
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

	// Wait for the right client who wants to connect - the game client.
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("failed to accept TCP connection: %w", err)
		}
		p.logger.Debug("Accepted new connection")

		// Recognise who is trying to connect by handling the initial data.
		if err := p.handleHandshake(conn); err != nil {
			p.logger.Debug("Handshake has failed")
			return err
		}

		p.logger.Debug("Successful handshake")
		break
	}

	if err := p.handleConnection(p.conn, onReceive); err != nil {
		p.logger.Error("Failed to handle connection", "error", err)
		return err
	}
	return nil
}

func (p *ListenerTCP) handleHandshake(conn TCPConn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		return fmt.Errorf("someone is already connected")
	}

	buf := make([]byte, 64)
	msg, err := p.readNext(conn, buf)
	if err != nil {
		return err
	}
	if !bytes.HasPrefix(msg, []byte{'#', '#'}) { // exactly `##username` of the connecting user
		return fmt.Errorf("invalid first packet, got: %s", string(msg))
	}

	p.conn = conn
	p.lastActive = time.Now()

	return nil
}

// handleConnection reads from the TCP connection and forwards the data received
// from the game client.
func (p *ListenerTCP) handleConnection(conn TCPConn, onReceive func(p []byte) (err error)) error {
	// Handle incoming data from the game client
	buf := make([]byte, 1024)

	for {
		clear(buf)

		msg, err := p.readNext(conn, buf)
		if err != nil {
			return err
		}

		// Mark when the last activity has happened
		p.lastActive = time.Now()

		if len(msg) == 0 {
			continue
		}

		p.logger.Debug("Received packet from the game client", "data", msg)

		if err := onReceive(msg); err != nil {
			p.logger.Warn("Failed to write data", logging.Error(err))
			return fmt.Errorf("failed to write to data channel: %w", err)
		}
	}
}

func (_ *ListenerTCP) readNext(conn TCPConn, buf []byte) ([]byte, error) {
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, nil
		}
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("game client has closed the TCP connection: %w", err)
		}
		if errors.Is(err, net.ErrClosed) {
			return nil, fmt.Errorf("tcp-listener has closed the connection: %w", err)
		}
		if errors.Is(err, io.ErrClosedPipe) {
			return nil, fmt.Errorf("tcp-listener has already closed the connection: %w", err)
		}

		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	return buf[:n], nil
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

	p.lastActive = time.Now()
	// p.logger.Debug("Sent to the game client", "size", n, "data", msg[:n])
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

func (p *ListenerTCP) Alive(now time.Time, timeout time.Duration) bool {
	p.mu.RLock()
	alive := !p.closed && p.conn != nil && p.lastActive.After(now.Add(-timeout))
	p.mu.RUnlock()
	return alive
}
