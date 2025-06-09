package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
)

// Ensure DialerTCP implements Redirect interface
var _ Redirect = (*DialerTCP)(nil)

type DialerTCP struct {
	conn   TCPConn
	logger *slog.Logger
}

// DialTCP establishes a TCP connection with the given IPv4 and port.
func DialTCP(ipv4 string, portNumber string) (*DialerTCP, error) {
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
		conn:   tcpConn,
		logger: logger,
	}, nil
}

// Run handles reading from TCP and forwards data received from the game client.
func (p *DialerTCP) Run(ctx context.Context, onReceive func(p []byte) (err error)) error {
	if p.conn == nil {
		return fmt.Errorf("tcp-dial: tcp connection is nil")
	}

	defer func() {
		p.logger.Info("Closing the TCP dialer")
	}()

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("tcp-dial: context canceled: %w", ctx.Err())

		default:
			clear(buf)

			p.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, err := p.conn.Read(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue
				}
				if err == io.EOF {
					p.logger.Info("Connection closed by server")
					return nil
				}
				p.logger.Error("Error reading from server", logging.Error(err))
				return err
			}

			p.logger.Debug("Received TCP message", "size", n)

			if err := onReceive(buf[:n]); err != nil {
				return fmt.Errorf("tcp-dial: failed to handle data received from the game client to: %w", err)
			}
		}
	}
}

// Write sends a message over the TCP connection to the game client.
func (p *DialerTCP) Write(msg []byte) (int, error) {
	n, err := p.conn.Write(msg)
	if err != nil {
		p.logger.Error("Failed to send message", logging.Error(err))
		return n, err
	}
	p.logger.Debug("Message sent", "size", n, "msg", msg)
	return n, nil
}

// Close terminates the TCP connection.
func (p *DialerTCP) Close() error {
	err := p.conn.Close()
	if err != nil {
		p.logger.Error("Failed to close TCP connection", logging.Error(err))
	}
	return err
}
