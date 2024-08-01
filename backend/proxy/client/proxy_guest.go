package client

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

type GuestProxy struct {
	connTCP net.Listener
	connUDP *net.UDPConn
}

func NewGuestProxyIP(ip net.IP) (*GuestProxy, error) {
	tcpAddr, udpAddr := net.JoinHostPort(ip.To4().String(), "6114"), net.JoinHostPort(ip.To4().String(), "6113")
	return NewGuestProxy(tcpAddr, udpAddr)
}

func NewGuestProxy(tcpAddr, udpAddr string) (*GuestProxy, error) {
	tcpListener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}

	srcAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, err
	}

	p := GuestProxy{}
	p.connTCP = tcpListener
	p.connUDP = srcConn
	return &p, nil
}

func (p *GuestProxy) Addr() string {
	return fmt.Sprintf("tcp=%s udp=%s", p.connTCP.Addr().String(), p.connUDP.LocalAddr().String())
}

func (p *GuestProxy) RunTCP(ctx context.Context, rw io.ReadWriteCloser) error {
	reader := func(conn net.Conn) {
		defer func() {
			slog.Warn("Closing connection to client", "protocol", "tcp")
			conn.Close()
			rw.Close()
		}()

		go func() {
			if _, err := io.Copy(conn, rw); err != nil {
				slog.Warn("Error copying from client to server", "error", err.Error(), "protocol", "tcp")
			}
			conn.Close()
		}()

		_, err := io.Copy(rw, conn)
		if err != nil {
			slog.Warn("Error copying from server to client", "error", err.Error(), "protocol", "tcp")
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
		slog.Debug("Accepted connection on port", "port", conn.RemoteAddr(), "protocol", "tcp")
		go reader(conn)
	}
}

func (p *GuestProxy) RunUDP(ctx context.Context, rw io.ReadWriteCloser) error {
	defer func() {
		log.Println(p.connUDP.Close())
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

			slog.Debug("Received from UDP", "payload", buf[0:n], "addr", addr, "protocol", "udp")

			if _, err := rw.Write(buf[:n]); err != nil {
				log.Println("(udp): Error writing to server: ", err)
				return err
			}
			slog.Debug("Wrote", "payload", buf[0:n], "addr", addr, "protocol", "udp")
		}
	})

	return g.Wait()
}
