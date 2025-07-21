package redirect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
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
	mu         sync.RWMutex
	conn       UDPConn
	OnReceive  ReceiveFunc
	logger     *slog.Logger
	lastActive time.Time
}

// NewDialUDP establishes the UDP connection with the given IPv4 and port.
// It can be used to connect to the game server of a guest peers.
func NewDialUDP(ipv4 string, portNumber string, onReceive ReceiveFunc) (*DialerUDP, error) {
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

	dialConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("dial-udp: could not dial over udp: %w", err)
	}

	log := slog.With(
		slog.String("redirect", "dial-udp"),
		slog.String("local", dialConn.LocalAddr().String()),
		slog.String("remote", dialConn.RemoteAddr().String()),
	)
	log.Info("Dialed via UDP")

	return &DialerUDP{
		conn:       dialConn,
		OnReceive:  onReceive,
		logger:     log,
		lastActive: time.Now(),
	}, nil
}

// Run reads UDP packets and calls the provided onReceive callback for each
// message received from the game client.
func (p *DialerUDP) Run(ctx context.Context) error {
	defer func() {
		if err := p.Close(); err != nil {
			p.logger.Error("Error during UDP connection close", logging.Error(err))
		}
	}()

	dialerConn := p.conn

	buf := make([]byte, 1024)
	for {
		if p.conn == nil {
			return fmt.Errorf("dial-udp: UDP connection is nil")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dialerConn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, _, err := dialerConn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					p.lastActive = time.Now()
					continue
				}
				p.logger.Warn("UDP read error", logging.Error(err))
				return fmt.Errorf("dial-udp: failed to read UDP message: %w", err)
			}

			p.lastActive = time.Now()

			if err := p.OnReceive(buf[:n]); err != nil {
				return fmt.Errorf("dial-udp: failed to handle data received from game client: %w", err)
			}
		}
	}
}

// Write sends a message over the UDP connection to the game client.
func (p *DialerUDP) Write(msg []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return 0, fmt.Errorf("dial-udp: UDP connection is nil")
	}
	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send UDP message", logging.Error(err))
		return n, fmt.Errorf("dial-udp: failed to write UDP message: %w", err)
	}
	p.lastActive = time.Now()
	return n, nil
}

// Close terminates the UDP connection.
func (p *DialerUDP) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return nil // Already closed or never opened
	}
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close UDP connection", logging.Error(err))
		return err
	}
	p.conn = nil // Prevent double close
	p.logger.Info("UDP connection closed")
	return nil
}

// Alive reports whether the UDP dialer is alive based on the last activity time and a timeout.
func (p *DialerUDP) Alive(now time.Time, timeout time.Duration) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.conn == nil {
		return false
	}
	return p.lastActive.After(now.Add(-timeout))
}
