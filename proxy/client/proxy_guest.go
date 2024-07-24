package client

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"
)

type GuestProxy struct {
	ExposedOnIP string

	OnUDPMessage func(ctx context.Context, handler func(msg []byte)) error
	OnTCPMessage func(ctx context.Context, handler func(msg []byte)) error

	WriteUDPMessage func(ctx context.Context, msg []byte) error
	WriteTCPMessage func(ctx context.Context, msg []byte) error
}

func NewGuestProxy(exposedIP string) *GuestProxy {
	p := GuestProxy{ExposedOnIP: exposedIP}
	slog.Info("Configured proxy", "proxyIP", p.ExposedOnIP)
	return &p
}

func (p *GuestProxy) Start(ctx context.Context) {
	go p.tcpAsHost(ctx)
	go p.udpAsHost(ctx)

	select {
	case <-ctx.Done():
		// p.Close()
		break
	}
}

func (p *GuestProxy) tcpAsHost(ctx context.Context) {
	slog.Info("Starting proxy for TCP")

	tcpListener, err := net.Listen("tcp", net.JoinHostPort(p.ExposedOnIP, "6114"))
	if err != nil {
		fmt.Println("Error listening on TCP:", err.Error())
		return
	}
	defer tcpListener.Close()
	fmt.Println("Listening TCP on", tcpListener.Addr().String())

	processPackets := func(ctx context.Context, clientConn net.Conn) {
		go p.OnTCPMessage(ctx, func(msg []byte) {
			_, err = clientConn.Write(msg)
			if err != nil {
				fmt.Println("(tcp): Error writing to client: ", err.Error())
			}
		})
		defer func() {
			log.Println("(tcp): Closed connection to client")
			clientConn.Close()
		}()

		for {
			if ctx.Err() != nil {
				return
			}

			buf := make([]byte, 1024)
			n, err := clientConn.Read(buf)
			if err != nil {
				fmt.Println("(tcp): Error reading from client: ", err.Error())
				return
			}
			p.WriteTCPMessage(ctx, buf[:n])
		}
	}

	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := tcpListener.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			continue
		}
		fmt.Println("(tcp): Accepted connection on port", conn.RemoteAddr())

		// TODO: Use workgroup
		go processPackets(ctx, conn)
	}
}

func (p *GuestProxy) udpAsHost(ctx context.Context) error {
	slog.Info("Starting proxy for UDP")

	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.ExposedOnIP, "6113"))
	if err != nil {
		fmt.Println("Error resolving UDP address: ", err.Error())
		return err
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		fmt.Println("Error listening on UDP: ", err.Error())
		return err
	}
	defer srcConn.Close()

	var (
		clientDestAddr *net.UDPAddr
		clientDestOnce sync.Once
	)
	setClientAddr := func(addr *net.UDPAddr) func() {
		return func() { clientDestAddr = addr }
	}

	go p.OnUDPMessage(ctx, func(msg []byte) {
		fmt.Println("(udp): (server): Received ", msg, " from ")

		if clientDestAddr != nil {
			srcConn.WriteToUDP(msg, clientDestAddr)
			fmt.Println("(udp): (server): wrote to client", msg)
		}
	})

	// Goroutine to forward source -> destination
	for {
		buf := make([]byte, 1024)
		n, addr, err := srcConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("(udp): Error reading from client: ", err)
			return err
		}
		clientDestOnce.Do(setClientAddr(addr))

		fmt.Println("(udp): (client): Received ", (buf[0:n]), " from ", addr)

		err = p.WriteUDPMessage(ctx, buf[0:n])
		if err != nil {
			fmt.Println("(udp): Error writing to server: ", err)
			return err
		}
		fmt.Println("(udp): (client): wrote to server", buf[0:n])
	}
}
