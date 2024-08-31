package p2p

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

var _ Redirector = (*DiallerTCP)(nil)

type DiallerTCP struct {
	tcpConn net.Conn
}

func DialTCP(ipv4 string, portNumber string) (*DiallerTCP, error) {
	if portNumber == "" {
		portNumber = "6114"
	}
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(ipv4, portNumber), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to game server on 6114: %s", err.Error())
	}
	slog.Info("Host: Connected TCP", "local", tcpConn.LocalAddr().String(), "remote", tcpConn.RemoteAddr().String())

	return &DiallerTCP{
		tcpConn: tcpConn,
	}, nil
}

func (p *DiallerTCP) Run(ctx context.Context, rw io.ReadWriteCloser) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if _, err := io.Copy(rw, p.tcpConn); err != nil {
			log.Println(err)
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
				log.Println("(tcp): Error reading from server: ", err)
				return err
			}

			slog.Debug("Received TCP message", "message", buf[0:n], "length", n)
			if _, err := rw.Write(buf[0:n]); err != nil {
				return err
			}
		}
	})

	return g.Wait()
}

func (p *DiallerTCP) Write(msg []byte) (n int, err error) {
	n, err = p.tcpConn.Write(msg)
	if err != nil {
		log.Println("(tcp): Error writing to server: ", err)
		return n, err
	}
	log.Println("(tcp): wrote to server", msg)
	return n, err
}

func (p *DiallerTCP) Close() error {
	// return p.tcpConn.Close()
	return nil
}
