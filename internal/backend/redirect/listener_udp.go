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

var _ Redirect = (*ListenerUDP)(nil)

type ListenerUDP struct {
	connUDP *net.UDPConn

	onceSetAddr sync.Once
	udpAddr     *net.UDPAddr
	logger      *slog.Logger
}

func ListenUDP(ipv4 string, portNumber string) (*ListenerUDP, error) {
	if portNumber == "" {
		portNumber = "6113"
	}
	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("listen-udp: could not resolve udp address: %w", err)
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, fmt.Errorf("listen-udp: could not listen UDP %w", err)
	}

	logger := slog.With(
		slog.String("redirect", "listen-tcp"),
		slog.String("addr", srcAddr.String()),
	)
	logger.Info("Listening UDP")

	p := ListenerUDP{
		connUDP: srcConn,
		logger:  logger,
	}
	return &p, nil
}

func (p *ListenerUDP) Run(ctx context.Context, rw io.Writer) error {
	defer func() {
		p.logger.Warn("Closing UDP listener", "try-error", p.connUDP.Close())
	}()

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			buf := make([]byte, 1024)
			n, remoteAddr, err := p.connUDP.ReadFromUDP(buf)
			if err != nil {
				p.logger.Warn("Failed to read", "error", err)
				return err
			}
			p.setAddr(remoteAddr)

			if _, err := rw.Write(buf[:n]); err != nil {
				p.logger.Warn("Failed to write", "error", err, "payload", buf[0:n])
				return err
			}

			p.logger.Debug("Wrote", "payload", buf[0:n])
		}
	})

	return g.Wait()
}

func (p *ListenerUDP) Write(msg []byte) (int, error) {
	if p.udpAddr == nil || p.connUDP == nil {
		return 0, io.EOF
	}
	// return p.connUDP.WriteToUDP(msg, p.udpAddr)
	return p.connUDP.WriteTo(msg, p.udpAddr)
}

func (p *ListenerUDP) setAddr(addr *net.UDPAddr) {
	p.udpAddr = addr
	p.logger = p.logger.With("remote-addr", addr)
}

func (p *ListenerUDP) Close() error {
	return p.connUDP.Close()
}
