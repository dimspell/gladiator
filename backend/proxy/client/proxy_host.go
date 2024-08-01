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
	connTCP net.Conn
	connUDP *net.UDPConn
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

	// udp:6113
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(gameServerIP, "6113"))
	if err != nil {
		return nil, fmt.Errorf("could not resolve UDP address on 6113 as a host: %s", err.Error())
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	return &HostListener{
		connTCP: tcpConn,
		connUDP: udpConn,
	}, nil
}

func (d *HostListener) RunReaderUDP(ctx context.Context, onPacket func(msg []byte)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.connUDP == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, addr, err := d.connUDP.ReadFromUDP(buf)
		if err != nil {
			return fmt.Errorf("could not read UDP message: %s", err.Error())
		}

		slog.Debug("Received UDP message", "message", buf[0:n], "length", n, "fromAddr", addr.String())
		onPacket(buf[0:n])
	}
}

func (d *HostListener) RunReaderTCP(ctx context.Context, onPacket func(msg []byte)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.connTCP == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, err := d.connTCP.Read(buf)
		if err != nil {
			log.Println("(tcp): Error reading from server: ", err)
			return err
		}

		log.Println("(tcp): (server): Received ", buf[0:n])
		onPacket(buf[0:n])
	}
}

func (d *HostListener) WriteUDPMessage(msg []byte) error {
	_, err := d.connUDP.Write(msg)
	if err != nil {
		return fmt.Errorf("could not write UDP message: %s", err.Error())
	}
	log.Println("(udp): wrote to server", msg)
	return nil
}

func (d *HostListener) WriteTCPMessage(msg []byte) error {
	_, err := d.connTCP.Write(msg)
	if err != nil {
		log.Println("(tcp): Error writing to server: ", err)
		return nil
	}
	log.Println("(tcp): wrote to server", msg)

	return nil
}

func (d *HostListener) Close() error {
	return errors.Join(d.connTCP.Close(), d.connUDP.Close())
}
