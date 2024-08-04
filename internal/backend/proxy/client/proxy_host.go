package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

type HostProxy struct {
	tcpConn net.Conn

	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
}

// DialHost establishes the connection to the real game server exposed by the
// DispelMulti.exe process. Do not forget to set up the writer and reader.
// Do not forget to close the connection when done.
func DialHost(gameServerIP string) (*HostProxy, error) {
	// tcp:6114
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(gameServerIP, "6114"), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to game server on 6114: %s", err.Error())
	}
	log.Println("Host: Connected TCP", tcpConn.LocalAddr().String(), tcpConn.RemoteAddr().String())

	// udp:6113
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(gameServerIP, "6113"))
	if err != nil {
		return nil, fmt.Errorf("could not resolve UDP address on 6113 as a host: %s", err.Error())
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Host: Connected UDP", udpConn.LocalAddr().String(), udpConn.RemoteAddr().String())

	return &HostProxy{
		tcpConn: tcpConn,
		udpAddr: udpAddr,
		udpConn: udpConn,
	}, nil
}

type Proxer interface {
	RunUDP(ctx context.Context, rw io.ReadWriteCloser) error
	RunTCP(ctx context.Context, rw io.ReadWriteCloser) error
	WriteUDPMessage(msg []byte) error
	WriteTCPMessage(msg []byte) error
}

func (p *HostProxy) RunUDP(ctx context.Context, rw io.ReadWriteCloser) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			buf := make([]byte, 1024)
			n, err := rw.Read(buf)
			if err != nil {
				log.Println(err)
				return err
			}
			if _, err := p.udpConn.WriteToUDP(buf[:n], p.udpAddr); err != nil {
				slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
				return err
			}
		}
	})
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

func (p *HostProxy) RunTCP(ctx context.Context, rw io.ReadWriteCloser) error {
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

func (p *HostProxy) WriteUDPMessage(msg []byte) error {
	_, err := p.udpConn.Write(msg)
	if err != nil {
		return fmt.Errorf("could not write UDP message: %s", err.Error())
	}
	log.Println("(udp): wrote to server", msg)
	return nil
}

func (p *HostProxy) WriteTCPMessage(msg []byte) error {
	_, err := p.tcpConn.Write(msg)
	if err != nil {
		log.Println("(tcp): Error writing to server: ", err)
		return nil
	}
	log.Println("(tcp): wrote to server", msg)
	return nil
}

func (p *HostProxy) Close() error {
	return errors.Join(p.tcpConn.Close(), p.udpConn.Close())
}
