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
}

func DialTCP(ipv4 string, portNumber string) (*DialerTCP, error) {
	if portNumber == "" {
		portNumber = "6114"
	}
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(ipv4, portNumber), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to game server on 6114: %s", err.Error())
	}
	slog.Info("Host: Connected TCP", "local", tcpConn.LocalAddr().String(), "remote", tcpConn.RemoteAddr().String())

	return &DialerTCP{
		tcpConn: tcpConn,
	}, nil
}

func (p *DialerTCP) Run(ctx context.Context, rw io.ReadWriteCloser) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if _, err := io.Copy(rw, p.tcpConn); err != nil {
			slog.Error("Could not copy the payload", "error", err)
			return err
		}
		return nil
	})
	g.Go(func() error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if p.tcpConn == nil {
				return io.EOF
			}

			buf := make([]byte, 1024)
			n, err := p.tcpConn.Read(buf)
			if err != nil {
				if err == io.EOF {
					slog.Debug("Connection closed to the game client", "proto", "tcp", "addr", p.tcpConn.LocalAddr())
					return err
				}
				slog.Debug("Error reading from server", "error", err, "proto", "tcp")
				return err
			}

			slog.Debug("Received TCP message", "message", buf[0:n], "length", n, "proto", "tcp")
			if _, err := rw.Write(buf[0:n]); err != nil {
				return err
			}
		}
	})

	return g.Wait()
}

func (p *DialerTCP) Write(msg []byte) (n int, err error) {
	n, err = p.tcpConn.Write(msg)
	if err != nil {
		slog.Error("(tcp): Error writing to server", "error", err)
		return n, err
	}
	slog.Debug("(tcp): wrote to server", "msg", msg)
	return n, err
}

func (p *DialerTCP) Close() error {
	return p.tcpConn.Close()
}
