package redirect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

var _ Redirect = (*DialerTCP)(nil)

type DialerTCP struct {
	tcpConn net.Conn
	logger  *slog.Logger
}

func DialTCP(ipv4 string, portNumber string) (*DialerTCP, error) {
	if portNumber == "" {
		portNumber = "6114"
	}
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(ipv4, portNumber), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to game server on 6114: %s", err.Error())
	}

	logger := slog.With(
		slog.String("redirect", "dial-tcp"),
		slog.String("local", tcpConn.LocalAddr().String()),
		slog.String("remote", tcpConn.RemoteAddr().String()),
	)
	logger.Info("Dialed via TCP")

	return &DialerTCP{
		tcpConn: tcpConn,
		logger:  logger,
	}, nil
}

func (p *DialerTCP) Run(ctx context.Context, dc io.Writer) error {
	if p.tcpConn == nil {
		return fmt.Errorf("tcp-dial: tcp connection is nil")
	}

	defer func() {
		p.logger.Info("Closing the TCP dialer")
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("tcp-dial: context canceled: %w", err)
			}

			buf := make([]byte, 1024)
			n, err := p.tcpConn.Read(buf)
			if err != nil {
				if err == io.EOF {
					p.logger.Debug("Connection closed to the game client")
					return err
				}
				p.logger.Debug("Error reading from server", "error", err)
				return err
			}

			p.logger.Debug("Received TCP message", "message", buf[0:n])

			if _, err := dc.Write(buf[0:n]); err != nil {
				return fmt.Errorf("tcp-dial: could not write to the datachannel: %w", err)
			}
		}
	})

	if err := g.Wait(); err != nil {
		p.logger.Debug("Error during game server", "error", err)
		return err
	}
	return nil
}

func (p *DialerTCP) Write(msg []byte) (n int, err error) {
	n, err = p.tcpConn.Write(msg)
	if err != nil {
		p.logger.Error("Could not send a message", "error", err)
		return n, err
	}
	p.logger.Debug("Wrote to proxy", "msg", msg)
	return n, err
}

func (p *DialerTCP) Close() error {
	return p.tcpConn.Close()
}
