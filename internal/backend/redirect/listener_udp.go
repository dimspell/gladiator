package redirect

import (
	"context"
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
	// writeUDP chan []byte
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
		// writeUDP: make(chan []byte),
		connUDP: srcConn,
	}
	return &p, nil
}

func (p *ListenerUDP) Run(ctx context.Context, rw io.Writer) error {
	defer func() {
		slog.Warn("Closing udp connnection", "error", p.connUDP.Close())
		// close(p.writeUDP)
	}()

	g, ctx := errgroup.WithContext(ctx)

	// g.Go(func() error {
	// 	for {
	// 		if ctx.Err() != nil {
	// 			return ctx.Err()
	// 		}
	//
	// 		buf := make([]byte, 1024)
	// 		n, err := rw.Read(buf)
	// 		if err != nil {
	// 			slog.Warn("Error reading from UDP", "error", err, "protocol", "udp")
	// 			return err
	// 		}
	//
	// 		slog.Debug("Received from RW", "payload", buf[0:n], "protocol", "udp")
	// 		if clientDestAddr != nil {
	// 			continue
	// 		}
	// 		if _, err := p.connUDP.WriteToUDP(buf[:n], clientDestAddr); err != nil {
	// 			slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
	// 			return err
	// 		}
	// 	}
	// })
	// g.Go(func() error {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return ctx.Err()
	// 		case msg, ok := <-p.writeUDP:
	// 			if !ok {
	// 				return fmt.Errorf("closed channel")
	// 			}
	// 			if clientDestAddr != nil {
	// 				continue
	// 			}
	// 			if _, err := p.connUDP.WriteToUDP(msg, clientDestAddr); err != nil {
	// 				slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
	// 				return err
	// 			}
	// 		}
	// 	}
	// })
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
			p.setAddr(addr)

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
	if p.udpAddr == nil || p.connUDP == nil {
		return 0, io.EOF
	}
	return p.connUDP.WriteToUDP(msg, p.udpAddr)
}

func (p *ListenerUDP) setAddr(addr *net.UDPAddr) {
	p.udpAddr = addr
}

func (p *ListenerUDP) Close() error {
	return p.connUDP.Close()
}
