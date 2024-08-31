package p2p

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

var _ Redirector = (*ListenerUDP)(nil)

type ListenerUDP struct {
	connUDP  *net.UDPConn
	writeUDP chan []byte
}

func ListenUDP(ipv4 string, portNumber string) (*ListenerUDP, error) {
	if portNumber == "" {
		portNumber = "6113"
	}
	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ipv4, portNumber))
	if err != nil {
		return nil, err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, err
	}
	slog.Info("Guest: Listening UDP", "addr", srcAddr.String())

	p := ListenerUDP{
		writeUDP: make(chan []byte),
		connUDP:  srcConn,
	}
	return &p, nil
}

func (p *ListenerUDP) Run(ctx context.Context, rw io.ReadWriteCloser) error {
	defer func() {
		log.Println(p.connUDP.Close())
		close(p.writeUDP)
	}()

	var (
		clientDestAddr *net.UDPAddr
		clientDestOnce sync.Once
		setClientAddr  = func(addr *net.UDPAddr) func() { return func() { clientDestAddr = addr } }
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			buf := make([]byte, 1024)
			n, err := rw.Read(buf)
			if err != nil {
				slog.Warn("Error reading from UDP", "error", err, "protocol", "udp")
				return err
			}

			slog.Debug("Received from RW", "payload", buf[0:n], "protocol", "udp")
			if clientDestAddr != nil {
				continue
			}
			if _, err := p.connUDP.WriteToUDP(buf[:n], clientDestAddr); err != nil {
				slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
				return err
			}
		}
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-p.writeUDP:
				if !ok {
					return fmt.Errorf("closed channel")
				}
				if clientDestAddr != nil {
					continue
				}
				if _, err := p.connUDP.WriteToUDP(msg, clientDestAddr); err != nil {
					slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
					return err
				}
			}
		}
	})
	g.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			buf := make([]byte, 1024)
			n, addr, err := p.connUDP.ReadFromUDP(buf)
			if err != nil {
				slog.Warn("Error reading from UDP ", "error", err, "protocol", "udp")
				return err
			}
			clientDestOnce.Do(setClientAddr(addr))

			if _, err := rw.Write(buf[:n]); err != nil {
				slog.Warn("Error writing to RW", "error", err, "protocol", "udp", "from", addr, "payload", buf[0:n])
				return err
			}
			slog.Debug("Redirected to RW", "payload", buf[0:n], "addr", addr, "protocol", "udp")
		}
	})

	return g.Wait()
}

func (p *ListenerUDP) Write(msg []byte) (int, error) {
	if p.connUDP == nil {
		return 0, io.EOF
	}
	p.writeUDP <- msg
	return len(msg), nil
}

func (p *ListenerUDP) Close() error {
	return p.connUDP.Close()
}
