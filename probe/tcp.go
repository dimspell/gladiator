package probe

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

type TCPChecker struct {
	Timeout time.Duration
	Address string
}

func (c *TCPChecker) Check() error {
	conn, err := net.DialTimeout("tcp", c.Address, c.Timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
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

		time.Sleep(3 * time.Second)

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
