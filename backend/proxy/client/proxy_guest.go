package client

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
)

type GuestProxy struct {
	ExposedOnIP string

	connTCP net.Listener
	connUDP *net.UDPConn
}

func NewGuestProxy(exposedIP string) (*GuestProxy, error) {
	p := GuestProxy{ExposedOnIP: exposedIP}
	slog.Info("Configured proxy", "proxyIP", p.ExposedOnIP)

	tcpListener, err := net.Listen("tcp", net.JoinHostPort(p.ExposedOnIP, "6114"))
	if err != nil {
		return nil, err
	}
	p.connTCP = tcpListener

	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.ExposedOnIP, "6113"))
	if err != nil {
		return nil, err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, err
	}
	p.connUDP = srcConn

	return &p, nil
}

func (p *GuestProxy) RunTCP(ctx context.Context, rw io.ReadWriteCloser) error {
	reader := func(conn net.Conn) {
		defer func() {
			log.Println("(tcp): Closed connection to client")
			conn.Close()
			rw.Close()
		}()

		go func() {
			if _, err := io.Copy(conn, rw); err != nil {
				log.Println("(tcp): Error copying from client to server: ", err.Error())
			}
			conn.Close()
		}()

		_, err := io.Copy(rw, conn)
		if err != nil {
			log.Println("(tcp): Error copying from server to client: ", err.Error())
		}
	}

	defer p.connTCP.Close()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		conn, err := p.connTCP.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}
		log.Println("(tcp): Accepted connection on port", conn.RemoteAddr())

		go reader(conn)
	}
}

func (p *GuestProxy) RunUDP(ctx context.Context, rw io.ReadWriteCloser) error {
	defer p.connUDP.Close()

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
				return err
			}

			if clientDestAddr != nil {
				continue
			}
			if _, err := p.connUDP.WriteToUDP(buf[:n], clientDestAddr); err != nil {
				return err
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
				log.Println("(udp): Error reading from client: ", err)
				return err
			}
			clientDestOnce.Do(setClientAddr(addr))

			log.Println("(udp): (client): Received ", buf[0:n], " from ", addr)

			if _, err := rw.Write(buf[:n]); err != nil {
				log.Println("(udp): Error writing to server: ", err)
				return err
			}
			log.Println("(udp): (client): wrote to server", buf[0:n])
		}
	})

	return g.Wait()
}
