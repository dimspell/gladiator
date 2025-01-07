package redirect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"
)

var _ Redirect = (*DialerUDP)(nil)

type DialerUDP struct {
	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
	logger  *slog.Logger
}

func DialUDP(ipv4 string, portNumber string) (*DialerUDP, error) {
	if portNumber == "" {
		portNumber = "6113"
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
		udpAddr: udpAddr,
		udpConn: udpConn,
		logger:  logger,
	}, nil
}

func (p *DialerUDP) Run(ctx context.Context, dc io.Writer) error {
	if p.udpConn == nil {
		return fmt.Errorf("dial-udp: udp connection is nil")
	}

	defer func() {
		p.logger.Info("Closing the UDP dialer")
	}()

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}

			buf := make([]byte, 1024)
			n, addr, err := p.udpConn.ReadFromUDP(buf)
			if err != nil {
				return fmt.Errorf("could not read UDP message: %s", err.Error())
			}

			p.logger.Debug("Received UDP message", "message", buf[0:n], "length", n, "fromAddr", addr.String())

			if _, err := dc.Write(buf[0:n]); err != nil {
				return err
			}
		}
	})
	g.Go(func() error {
		<-ctx.Done()
		p.udpConn.Close()
		return ctx.Err()
	})

	return g.Wait()
}

func (p *DialerUDP) Write(msg []byte) (n int, err error) {
	n, err = p.udpConn.Write(msg)
	if err != nil {
		return n, fmt.Errorf("dial-udp: could not write UDP message: %w", err)
	}
	p.logger.Debug("Wrote to proxy", "msg", msg)
	return n, err
}

func (p *DialerUDP) Close() error {
	return p.udpConn.Close()
}
