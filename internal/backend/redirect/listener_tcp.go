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
	listenerTCP net.Listener
	logger      *slog.Logger

	mu      sync.RWMutex
	connTCP net.Conn
}

func ListenTCP(ipv4 string, portNumber string) (*ListenerTCP, error) {
	if net.ParseIP(ipv4) == nil {
		return nil, fmt.Errorf("listen-tcp: invalid IPv4 address format")
	}

	if portNumber == "" {
		portNumber = defaultTcpPort
	}

	tcpListener, err := net.Listen("tcp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, err
	}

	logger := slog.With(
		slog.String("redirect", "listen-tcp"),
		slog.String("addr", tcpListener.Addr().String()),
	)
	logger.Info("Listening TCP")

	return &ListenerTCP{
		listenerTCP: tcpListener,
		logger:      logger,
	}, nil
}

func (p *ListenerTCP) Run(ctx context.Context, rw io.Writer) error {
	defer func() {
		p.logger.Info("Closing listener")

		if err := p.listenerTCP.Close(); err != nil {
			p.logger.Error("Could not close listener", "error", err)
			return
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := p.listenerTCP.Accept()
			if err != nil {
				p.logger.Error("Failed to accept TCP connection", "error", err)
				continue
			}
			p.logger.Debug("Accepted connection", "remote-addr", conn.RemoteAddr())

			p.mu.Lock()
			p.connTCP = conn
			p.mu.Unlock()

			go func() {
				if err := p.handleConnection(ctx, conn, rw); err != nil {
					p.logger.Error("Error handling connection", "error", err)
				}
			}()
		}
	}
}

func (p *ListenerTCP) handleConnection(ctx context.Context, conn net.Conn, dc io.Writer) error {
	defer func() {
		p.mu.Lock()
		p.connTCP = nil
		p.mu.Unlock()

		p.logger.Debug("Closing TCP connection", "remote-addr", conn.RemoteAddr())
		if err := conn.Close(); err != nil {
			p.logger.Warn("Error closing TCP connection", "error", err)
		}
	}()

	// Handle incoming data
	buffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buffer)

			n, err := conn.Read(buffer)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					p.logger.Warn("Error reading from TCP", "error", err)
				}
				return err
			}

			if _, err := dc.Write(buffer[:n]); err != nil {
				p.logger.Warn("Error writing to TCP", "error", err)
				return err
			}

			p.logger.Debug("Received from client", "payload", buffer[0:n])
		}
	}
}

func (p *ListenerTCP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.connTCP == nil {
		return 0, fmt.Errorf("listener-tcp: no active TCP connection")
	}
	return p.connTCP.Write(msg)
}

func (p *ListenerTCP) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errConn, errListener error

	// Close an existing TCP connection if any
	if p.connTCP != nil {
		errConn = p.connTCP.Close()
		p.connTCP = nil
	}

	// Close the TCP listener
	errListener = p.listenerTCP.Close()
	p.listenerTCP = nil

	return errors.Join(errConn, errListener)
}
