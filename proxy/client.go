package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sync/errgroup"
)

type ClientProxy struct {
	HostIP   string
	MasterIP string

	ConnectionTimeout time.Duration
}

func NewClientProxy(masterIP string) *ClientProxy {
	p := ClientProxy{
		HostIP:            "127.0.1.28",
		MasterIP:          masterIP,
		ConnectionTimeout: DefaultConnectionTimeout,
	}
	slog.Info("Configured proxy", "masterIP", p.MasterIP, "proxyIP", p.HostIP)
	return &p
}

func (p *ClientProxy) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		p.tcpAsHost(ctx)
		return fmt.Errorf("tcpAsHost stopped")
	})
	g.Go(func() error {
		return p.udpAsHost(ctx)
	})
	return g.Wait()
}

func (p *ClientProxy) tcpAsHost(ctx context.Context) {
	slog.Info("Starting proxy for TCP")

	tcpListener, err := net.Listen("tcp", net.JoinHostPort(p.HostIP, "6114"))
	if err != nil {
		fmt.Println("Error listening on TCP:", err.Error())
		return
	}
	defer tcpListener.Close()
	fmt.Println("Listening TCP on", tcpListener.Addr().String())

	serverConn, err := net.DialTimeout("tcp", net.JoinHostPort(p.MasterIP, "6114"), p.ConnectionTimeout)
	if err != nil {
		return
	}
	fmt.Println("(tcp): conencted to ", serverConn.RemoteAddr())
	defer serverConn.Close()

	processPackets := func(ctx context.Context, clientConn net.Conn) {
		defer func() {
			log.Println("(tcp): Closed connection to client")
			clientConn.Close()
		}()
		defer func() {
			log.Println("(tcp): Closed connection to server")
			serverConn.Close()
		}()

		go func() {
			_, err = io.Copy(clientConn, serverConn)
			if err != nil {
				fmt.Println("(tcp): Error copying from client to server: ", err.Error())
			}
			clientConn.Close()
		}()

		_, err = io.Copy(serverConn, clientConn)
		if err != nil {
			fmt.Println("(tcp): Error copying from server to client: ", err.Error())
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

func (p *ClientProxy) udpAsHost(ctx context.Context) error {
	slog.Info("Starting proxy for UDP")

	srcAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.HostIP, "6113"))
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

	var clientDest *net.UDPAddr

	destAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.MasterIP, "6113"))
	if err != nil {
		fmt.Println("Error resolving UDP address: ", err.Error())
		return err
	}
	destConn, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer destConn.Close()

	// Goroutine to forward destination -> source
	go func() {
		for {
			buf := make([]byte, 1024)
			n, addr, err := destConn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("(udp): Error reading from server: ", err)
				return
			}

			fmt.Println("(udp): (server): Received ", (buf[0:n]), " from ", addr, clientDest)

			// clientDestConn, err := net.DialUDP("udp", nil, clientDest)
			// if err != nil {
			// 	log.Fatal(err)
			// }

			srcConn.WriteToUDP(buf[0:n], clientDest)

			// _, err = clientDestConn.Write(buf[0:n])
			if err != nil {
				fmt.Println("(udp): Error writing to client: ", err)
				continue
			}
			fmt.Println("(udp): (server): wrote to client", buf[0:n])
		}
	}()

	// Goroutine to forward source -> destination
	for {
		buf := make([]byte, 1024)
		n, addr, err := srcConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("(udp): Error reading from client: ", err)
			continue
		}
		clientDest = addr
		fmt.Println("Client dest =", clientDest)

		fmt.Println("(udp): (client): Received ", (buf[0:n]), " from ", addr)

		_, err = destConn.Write(buf[0:n])
		if err != nil {
			fmt.Println("(udp): Error writing to server: ", err)
			continue
		}
		fmt.Println("(udp): (client): wrote to server", buf[0:n])
	}
	return nil
}
