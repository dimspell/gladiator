package redirect

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"

	"golang.org/x/sync/errgroup"
)

var _ Redirect = (*DialerUDP)(nil)

type DialerUDP struct {
	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
}

func DialUDP(ipv4 string, portNumber string) (*DialerUDP, error) {
	if portNumber == "" {
		portNumber = "6113"
	}
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, fmt.Errorf("could not resolve UDP address on 6113 as a host: %s", err.Error())
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Host: Connected UDP", "local", udpConn.LocalAddr().String(), "remote", udpConn.RemoteAddr().String())

	return &DialerUDP{
		udpAddr: udpAddr,
		udpConn: udpConn,
	}, nil
}

func (p *DialerUDP) Run(ctx context.Context, rw io.Writer) error {
	g, ctx := errgroup.WithContext(ctx)

	// g.Go(func() error {
	// 	for {
	// 		buf := make([]byte, 1024)
	// 		n, err := rw.Read(buf)
	// 		if err != nil {
	// 			slog.Error("Failed to read", "error", err)
	// 			return err
	// 		}
	// 		if _, err := p.udpConn.WriteToUDP(buf[:n], p.udpAddr); err != nil {
	// 			slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
	// 			return err
	// 		}
	// 	}
	// })
	g.Go(func() error {
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if p.udpConn == nil {
				return io.EOF
			}

			buf := make([]byte, 1024)
			n, addr, err := p.udpConn.ReadFromUDP(buf)
			if err != nil {
				return fmt.Errorf("could not read UDP message: %s", err.Error())
			}

			slog.Debug("Received UDP message", "message", buf[0:n], "length", n, "fromAddr", addr.String())

			if _, err := rw.Write(buf[0:n]); err != nil {
				return err
			}
		}
	})

	return g.Wait()
}

func (p *DialerUDP) Write(msg []byte) (n int, err error) {
	n, err = p.udpConn.Write(msg)
	if err != nil {
		return n, fmt.Errorf("could not write UDP message: %s", err.Error())
	}
	slog.Debug("(udp): wrote to server", "msg", msg)
	return n, err
}

func (p *DialerUDP) Close() error {
	return p.udpConn.Close()
}
