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

type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	Write(b []byte) (n int, err error)
	WriteTo(b []byte, addr net.Addr) (int, error)
	Close() error
	SetReadDeadline(t time.Time) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// DialerUDP wraps the UDP connection used to communicate with a remote game
// server.
type DialerUDP struct {
	conn       UDPConn
	logger     *slog.Logger
	lastActive time.Time
}

// DialUDP establishes the UDP connection with the given IPv4 and port.
// It can be used to connect to the game server of a guest peers.
func DialUDP(ipv4 string, portNumber string) (*DialerUDP, error) {
	if net.ParseIP(ipv4) == nil {
		return nil, fmt.Errorf("dial-udp: invalid IPv4 address format")
	}

	if portNumber == "" {
		portNumber = defaultUDPPort
	}

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("dial-udp: could not resolve UDP address: %w", err)
	}

	rawConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial-udp: could not dial over udp: %w", err)
	}

	log := slog.With(
		slog.String("redirect", "dial-udp"),
		slog.String("local", rawConn.LocalAddr().String()),
		slog.String("remote", rawConn.RemoteAddr().String()),
	)
	log.Info("Dialed via UDP")

	return &DialerUDP{
		conn:       rawConn,
		logger:     log,
		lastActive: time.Now(),
	}, nil
}

// Run reads UDP packets and calls the provided onReceive callback for each
// message received from the game client.
func (p *DialerUDP) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	defer func() {
		_ = p.Close()
	}()

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("dial-udp: %w", ctx.Err())

		default:
			clear(buf)

			p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, _, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					p.lastActive = time.Now()
					continue
				}
				p.logger.Error("Error reading from UDP server", logging.Error(err))
				return fmt.Errorf("dial-udp: failed to read UDP message: %w", err)
			}

			p.lastActive = time.Now()
			// p.logger.Debug("Received UDP message", slog.Int("size", n)))

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
	// p.logger.Debug("Message sent", "size", n, "msg", msg)
	return n, nil
}

// Close terminates the UDP connection.
func (p *DialerUDP) Close() error {
	err := p.conn.Close()
	if err != nil {
		p.logger.Debug("Failed to close UDP connection", logging.Error(err))
	}
	return err
}
