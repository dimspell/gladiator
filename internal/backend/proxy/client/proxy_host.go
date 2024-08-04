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
)

type HostListener struct {
	tcpConn net.Conn

	udpAddr *net.UDPAddr
	udpConn *net.UDPConn
}

// ListenHost establishes the connection to the real game server exposed by the
// DispelMulti.exe process. Do not forget to set up the writer and reader.
// Do not forget to close the connection when done.
func ListenHost(gameServerIP string) (*HostListener, error) {
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

	return &HostListener{
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

func (d *HostListener) RunUDP(ctx context.Context, rw io.ReadWriteCloser) error {
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := rw.Read(buf)
			if err != nil {
				log.Println(err)
				return
			}
			if _, err := d.udpConn.WriteToUDP(buf[:n], d.udpAddr); err != nil {
				slog.Warn("Error writing to UDP", "error", err, "protocol", "udp")
				return
			}
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.udpConn == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, addr, err := d.udpConn.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("could not read UDP message: %s", err.Error())
		}

		slog.Debug("Received UDP message", "message", buf[0:n], "length", n, "fromAddr", addr.String())

		if _, err := rw.Write(buf[0:n]); err != nil {
			return err
		}
	}
}

func (d *HostListener) RunTCP(ctx context.Context, rw io.ReadWriteCloser) error {
	go func() {
		if _, err := io.Copy(rw, d.tcpConn); err != nil {
			log.Println(err)
			return
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.tcpConn == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, err := d.tcpConn.Read(buf)
		if err != nil {
			log.Println("(tcp): Error reading from server: ", err)
			return err
		}

		slog.Debug("Received TCP message", "message", buf[0:n], "length", n)
		if _, err := rw.Write(buf[0:n]); err != nil {
			return err
		}
	}
}

func (d *HostListener) WriteUDPMessage(msg []byte) error {
	_, err := d.udpConn.Write(msg)
	if err != nil {
		return fmt.Errorf("could not write UDP message: %s", err.Error())
	}
	log.Println("(udp): wrote to server", msg)
	return nil
}

func (d *HostListener) WriteTCPMessage(msg []byte) error {
	_, err := d.tcpConn.Write(msg)
	if err != nil {
		log.Println("(tcp): Error writing to server: ", err)
		return nil
	}
	log.Println("(tcp): wrote to server", msg)

	return nil
}

func (d *HostListener) Close() error {
	return errors.Join(d.tcpConn.Close(), d.udpConn.Close())
}
