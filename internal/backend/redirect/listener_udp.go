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

// Ensure ListenerUDP implements Redirect interface
var _ Redirect = (*ListenerUDP)(nil)

type ListenerUDP struct {
	logger *slog.Logger

	conn       UDPConn
	remoteAddr *net.UDPAddr
}

// ListenUDP initializes the UDP listener on the given IP and port.
func ListenUDP(ipv4 string, portNumber string) (*ListenerUDP, error) {
	if portNumber == "" {
		portNumber = defaultUDPPort
	}
	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("listen-udp: failed to resolve address: %w", err)
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, fmt.Errorf("listen-udp: failed to listen on UDP: %w", err)
	}

	logger := slog.With(
		slog.String("redirect", "listen-udp"),
		slog.String("remoteAddr", srcAddr.String()),
	)
	logger.Info("UDP listener started")

	p := ListenerUDP{
		conn:   srcConn,
		logger: logger,
	}
	return &p, nil
}

// Run listens for incoming UDP messages from the game client and forwards them.
func (p *ListenerUDP) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	go func() {
		<-ctx.Done()
		p.Close()
	}()

	defer func() {
		if err := p.conn.Close(); err != nil {
			p.logger.Error("Error closing UDP listener", logging.Error(err))
		}
	}()

	// Goroutine to read incoming messages
	buf := make([]byte, 1024)

	var onceSet sync.Once

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			clear(buf)

			p.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, remoteAddr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
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

			// Set remote address once (used for sending messages back)
			onceSet.Do(func() { p.remoteAddr = remoteAddr })

			// Forward the received message
			if err := onReceive(buf[:n]); err != nil {
				p.logger.Warn("Failed to write message", logging.Error(err), "payload", buf[:n])
				return fmt.Errorf("listen-udp: write error: %w", err)
			}

			// p.logger.Debug("Received UDP message", "size", n, "data", buf[:n])
		}
	}
}

// Write sends data to the last received address - to the game server.
func (p *ListenerUDP) Write(msg []byte) (int, error) {
	if p.remoteAddr == nil || p.conn == nil {
		return 0, fmt.Errorf("listen-udp: no remote address set")
	}

	n, err := p.conn.WriteTo(msg, p.remoteAddr)
	if err != nil {
		p.logger.Warn("Failed to send UDP message", logging.Error(err))
		return n, fmt.Errorf("listen-udp: send failed: %w", err)
	}

	// p.logger.Debug("Sent UDP message", "size", n, "data", msg)
	return n, nil
}

// Close shuts down the UDP listener.
func (p *ListenerUDP) Close() error {
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close UDP connection", logging.Error(err))
		return fmt.Errorf("listen-udp: close error: %w", err)
	}
	p.logger.Info("UDP listener closed")
	return nil
}
