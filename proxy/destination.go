package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
)

type Destination interface {
	OnUDPMessage(ctx context.Context, handler func(msg []byte)) error
	OnTCPMessage(ctx context.Context, handler func(msg []byte)) error

	WriteUDPMessage(ctx context.Context, msg []byte) error
	WriteTCPMessage(ctx context.Context, msg []byte) error

	Close()
}

var _ Destination = (*LocalDest)(nil)

type LocalDest struct {
	tcpConn net.Conn
	udpConn *net.UDPConn
}

func NewLocalDest(masterIP string) (*LocalDest, error) {
	tcpConn, err := net.DialTimeout("tcp", net.JoinHostPort(masterIP, "6114"), DefaultConnectionTimeout)
	if err != nil {
		return nil, err
	}
	fmt.Println("(tcp): connected to ", tcpConn.RemoteAddr())

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(masterIP, "6113"))
	if err != nil {
		fmt.Println("Error resolving UDP address: ", err.Error())
		return nil, err
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	return &LocalDest{
		tcpConn: tcpConn,
		udpConn: udpConn,
	}, nil
}

func (d *LocalDest) OnUDPMessage(ctx context.Context, handler func(msg []byte)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.udpConn == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, _, err := d.udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("(udp): Error reading from server: ", err)
			return err
		}

		fmt.Println("(udp): (server): Received ", buf[0:n])
		handler(buf[0:n])
	}
}

func (d *LocalDest) OnTCPMessage(ctx context.Context, handler func(msg []byte)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.udpConn == nil {
			return io.EOF
		}

		buf := make([]byte, 1024)
		n, err := d.tcpConn.Read(buf)
		if err != nil {
			fmt.Println("(tcp): Error reading from server: ", err)
			return err
		}

		fmt.Println("(tcp): (server): Received ", buf[0:n])
		handler(buf[0:n])
	}
}

func (d *LocalDest) WriteUDPMessage(ctx context.Context, msg []byte) error {
	_, err := d.udpConn.Write(msg)
	if err != nil {
		fmt.Println("(udp): Error writing to server: ", err)
		return nil
	}
	fmt.Println("(udp): wrote to server", msg)
	return nil
}

func (d *LocalDest) WriteTCPMessage(ctx context.Context, msg []byte) error {
	_, err := d.tcpConn.Write(msg)
	if err != nil {
		fmt.Println("(tcp): Error writing to server: ", err)
		return nil
	}
	fmt.Println("(tcp): wrote to server", msg)

	return nil
}

func (d *LocalDest) Close() {
	// TODO: Use multierr and return an error
	if d.tcpConn != nil {
		d.tcpConn.Close()
		d.tcpConn = nil
	}
	if d.udpConn != nil {
		d.udpConn.Close()
		d.udpConn = nil
	}
}
