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
	connTCP net.Listener
	connUDP *net.UDPConn

	writeUDP chan []byte
	writeTCP chan []byte
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
	log.Println("Guest: Listening TCP", tcpListener.Addr().String())

	srcAddr, err := net.ResolveUDPAddr("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return nil, err
	}
	log.Println("Guest: Listening UDP", srcConn.LocalAddr(), srcConn.RemoteAddr())

	p := GuestProxy{
		writeUDP: make(chan []byte),
		writeTCP: make(chan []byte),
	}
	p.connTCP = tcpListener
	p.connUDP = srcConn
	return &p, nil
}

func (p *GuestProxy) RunTCP(ctx context.Context, rw io.ReadWriteCloser) error {
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
		go p.readerTCP(ctx, conn, rw)
	}
}

func (p *GuestProxy) readerTCP(ctx context.Context, conn net.Conn, rw io.ReadWriteCloser) {
	defer func() {
		slog.Warn("Closing connection to client", "protocol", "tcp")
		conn.Close()
		rw.Close()
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if _, err := io.Copy(conn, rw); err != nil {
			slog.Warn("Error copying from client to server", "error", err, "protocol", "tcp")
			return err
		}
		return nil
	})
	g.Go(func() error {
		_, err := io.Copy(rw, conn)
		if err != nil {
			slog.Warn("Error copying from server to client", "error", err, "protocol", "tcp")
		}
		return err
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg := <-p.writeTCP:
				if _, err := conn.Write(msg); err != nil {
					slog.Warn("Error writing to TCP", "error", err, "protocol", "tcp")
					return err
				}
			}
		}
	})

	if err := g.Wait(); err != nil {
		slog.Error("Failed proxying TCP", "error", err)
	}
}

func (p *GuestProxy) RunUDP(ctx context.Context, rw io.ReadWriteCloser) error {
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
			case msg := <-p.writeUDP:
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

func (p *GuestProxy) WriteUDPMessage(msg []byte) error {
	if p.connUDP == nil {
		return io.EOF
	}
	p.writeUDP <- msg
	return nil
}

func (p *GuestProxy) WriteTCPMessage(msg []byte) error {
	if p.connTCP == nil {
		return io.EOF
	}
	p.writeTCP <- msg
	return nil
}
