package redirect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
)

// Ensure DialerUDP implements Redirect interface
var _ Redirect = (*DialerUDP)(nil)

type DialerUDP struct {
	addr   *net.UDPAddr
	conn   *net.UDPConn
	logger *slog.Logger
}

// DialUDP establishes a UDP connection with the given IPv4 and port.
func DialUDP(ipv4 string, portNumber string) (*DialerUDP, error) {
	if net.ParseIP(ipv4) == nil {
		return nil, fmt.Errorf("dial-udp: invalid IPv4 address format")
	}

	if portNumber == "" {
		portNumber = defaultUdpPort
	}

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("dial-udp: could not resolve UDP address: %w", err)
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial-udp: could not dial over udp: %w", err)
	}

	logger := slog.With(
		slog.String("redirect", "dial-udp"),
		slog.String("local", udpConn.LocalAddr().String()),
		slog.String("remote", udpConn.RemoteAddr().String()),
	)
	logger.Info("Dialed via UDP")

	return &DialerUDP{
		addr:   udpAddr,
		conn:   udpConn,
		logger: logger,
	}, nil
}

// Run handles reading from UDP and forwards data received from the game client.
func (p *DialerUDP) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	if p.conn == nil {
		return fmt.Errorf("dial-udp: udp connection is nil")
	}

	defer func() {
		p.logger.Info("Closing the UDP dialer")
		p.Close()
	}()

	buf := make([]byte, 1024) // Reuse buffer for efficiency
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("dial-udp: context canceled: %w", ctx.Err())
		default:
			clear(buf)

			p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, addr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue
				}
				p.logger.Error("Error reading from UDP server", logging.Error(err))
				return fmt.Errorf("dial-udp: failed to read UDP message: %w", err)
			}

			p.logger.Debug("Received UDP message", "size", n, "from", addr.String())

			if err := onReceive(buf[:n]); err != nil {
				return fmt.Errorf("dial-udp: failed to handle data received from game client: %w", err)
			}
		}
	}
}

// Write sends a message over the UDP connection to the game client.
func (p *DialerUDP) Write(msg []byte) (int, error) {
	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send UDP message", logging.Error(err))
		return n, fmt.Errorf("dial-udp: failed to write UDP message: %w", err)
	}
	p.logger.Debug("Message sent", "size", n, "msg", msg)
	return n, nil
}

// Close terminates the UDP connection.
func (p *DialerUDP) Close() error {
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close UDP connection", logging.Error(err))
	}
	return err
}
