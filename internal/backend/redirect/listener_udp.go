package redirect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Ensure ListenerUDP implements Redirect interface
var _ Redirect = (*ListenerUDP)(nil)

type ListenerUDP struct {
	conn *net.UDPConn

	onceSetAddr sync.Once
	addr        *net.UDPAddr

	logger *slog.Logger
}

// ListenUDP initializes the UDP listener on the given IP and port.
func ListenUDP(ipv4 string, portNumber string) (*ListenerUDP, error) {
	if portNumber == "" {
		portNumber = "6113"
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
		slog.String("redirect", "listen-tcp"),
		slog.String("addr", srcAddr.String()),
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
	defer func() {
		p.logger.Info("Closing UDP listener")
		if err := p.conn.Close(); err != nil {
			p.logger.Error("Error closing UDP listener", "error", err)
		}
	}()

	g, ctx := errgroup.WithContext(ctx)

	// Goroutine to read incoming messages
	g.Go(func() error {
		buf := make([]byte, 1024)

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				clear(buf)

				n, remoteAddr, err := p.conn.ReadFromUDP(buf)
				if err != nil {
					p.logger.Warn("Failed to read UDP message", "error", err)
					return fmt.Errorf("listen-udp: read error: %w", err)
				}

				// Set remote address once (used for sending messages back)
				p.onceSetAddr.Do(func() {
					p.setRemoteAddr(remoteAddr)
				})

				// Forward the received message
				if err := onReceive(buf[:n]); err != nil {
					p.logger.Warn("Failed to write message", "error", err, "payload", buf[:n])
					return fmt.Errorf("listen-udp: write error: %w", err)
				}

				p.logger.Debug("Received UDP message", "size", n, "data", buf[:n])
			}
		}
	})

	return g.Wait()
}

// Write sends the UDP packet to the game client (stored remote address).
func (p *ListenerUDP) Write(msg []byte) (int, error) {
	if p.addr == nil || p.conn == nil {
		return 0, fmt.Errorf("listen-udp: no remote address set")
	}

	n, err := p.conn.WriteTo(msg, p.addr)
	if err != nil {
		p.logger.Warn("Failed to send UDP message", "error", err)
		return n, fmt.Errorf("listen-udp: send failed: %w", err)
	}

	p.logger.Debug("Sent UDP message", "size", n, "data", msg)
	return n, nil
}

// setRemoteAddr safely sets the remote address for writing.
func (p *ListenerUDP) setRemoteAddr(addr *net.UDPAddr) {
	p.addr = addr
	p.logger = p.logger.With("remote-addr", addr.String())
	p.logger.Info("Remote UDP address set", "address", addr.String())
}

// Close shuts down the UDP listener.
func (p *ListenerUDP) Close() error {
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close UDP connection", "error", err)
		return fmt.Errorf("listen-udp: close error: %w", err)
	}
	p.logger.Info("UDP listener closed")
	return nil
}
