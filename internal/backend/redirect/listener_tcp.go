package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
)

var _ Redirect = (*ListenerTCP)(nil)

type ListenerTCP struct {
	listener net.Listener
	logger   *slog.Logger

	mu   sync.RWMutex
	conn net.Conn
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

// Run listens for incoming TCP connections and handles them.
func (p *ListenerTCP) Run(ctx context.Context, rw io.Writer) error {
	defer func() {
		p.logger.Info("Shutting down TCP listener")
		if err := p.listener.Close(); err != nil {
			p.logger.Error("Error closing listener", "error", err)
		}
	}()

	// Goroutine to handle shutdown on context cancellation
	go func() {
		<-ctx.Done()
		p.logger.Info("Listener shutting down due to context cancellation")
		p.listener.Close()
	}()

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return ctx.Err() // Exit gracefully on context cancellation
			default:
				p.logger.Error("Failed to accept TCP connection", "error", err)
				continue
			}
		}

		p.logger.Debug("Accepted new connection", "remote-addr", conn.RemoteAddr())

		// Store the latest active connection
		p.mu.Lock()
		p.conn = conn
		p.mu.Unlock()

		go func() {
			if err := p.handleConnection(ctx, conn, rw); err != nil {
				p.logger.Error("Error handling connection", "error", err)
			}
		}()
	}
}

// handleConnection reads from the TCP connection and writes to the provided io.Writer.
func (p *ListenerTCP) handleConnection(ctx context.Context, conn net.Conn, dc io.Writer) error {
	defer func() {
		p.mu.Lock()
		p.conn = nil
		p.mu.Unlock()

		p.logger.Debug("Closing TCP connection", "remote-addr", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			p.logger.Warn("Error closing TCP connection", "error", err)
		}
	}()

	// Handle incoming data
	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buf)

			n, err := conn.Read(buf)
			if err != nil {
				if errors.Is(err, io.EOF) {
					p.logger.Info("Client closed the connection")
					return nil
				}
				p.logger.Warn("Error reading from TCP connection", "error", err)
				return fmt.Errorf("listener-tcp: failed to read data: %w", err)
			}

			p.logger.Debug("Received data", "size", n, "data", buf[:n])

			if _, err := dc.Write(buf[:n]); err != nil {
				p.logger.Warn("Failed to write data", "error", err)
				return fmt.Errorf("listener-tcp: failed to write to data channel: %w", err)
			}
		}
	}
}

// Write sends data to the active TCP connection.
func (p *ListenerTCP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return 0, fmt.Errorf("listener-tcp: no active connection")
	}

	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send data", "error", err)
		return n, fmt.Errorf("listener-tcp: write failed: %w", err)
	}

	p.logger.Debug("Sent data", "size", n, "data", msg[:n])
	return n, nil
}

// Close shuts down the listener and any active connection.
func (p *ListenerTCP) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errConn, errListener error

	// Close active TCP connection if present
	if p.conn != nil {
		errConn = p.conn.Close()
		p.conn = nil
	}

	// Close the TCP listener
	errListener = p.listener.Close()
	p.listener = nil

	if errConn != nil || errListener != nil {
		return errors.Join(errConn, errListener)
	}

	p.logger.Info("TCP listener closed")
	return nil
}
