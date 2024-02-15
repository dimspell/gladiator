package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

var DefaultConnectionTimeout = 5 * time.Second

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
	go p.FakeHost(ctx)

	<-ctx.Done()
	return nil
}

func (p *ClientProxy) FakeHost(ctx context.Context) error {
	slog.Info("Started proxy")

	tcpListener, err := net.Listen("tcp", net.JoinHostPort(p.HostIP, "6114"))
	if err != nil {
		return err
	}
	defer tcpListener.Close()

	go p.tcpAsHost(ctx, tcpListener)

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(p.HostIP, "6113"))
	if err != nil {
		return err
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	go p.udpAsHost(ctx, udpConn)

	<-ctx.Done()
	return nil
}

func (p *ClientProxy) tcpAsHost(ctx context.Context, tcpListener net.Listener) {
	processPackets := func(ctx context.Context, clientConn net.Conn) {
		// defer clientConn.Close()

		serverConn, err := net.DialTimeout("tcp", net.JoinHostPort(p.MasterIP, "6114"), p.ConnectionTimeout)
		if err != nil {
			return
		}
		// defer serverConn.Close()

		go func() {
			_, _ = io.Copy(serverConn, clientConn)
			clientConn.Close()
		}()

		_, _ = io.Copy(clientConn, serverConn)
		serverConn.Close()
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
		fmt.Println("Accepted connection on port", conn.RemoteAddr())

		// TODO: Use workgroup
		go processPackets(ctx, conn)
	}
}

func (p *ClientProxy) udpAsHost(ctx context.Context, udpConn *net.UDPConn) {
	go func() {
		buf := make([]byte, 1024)
		for {
			n, addr, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			fmt.Println("Received ", string(buf[0:n]), " from ", addr)

			serverConn, err := net.Dial("udp", net.JoinHostPort(p.MasterIP, "6114"))
			if err != nil {
				return
			}
			defer serverConn.Close()

			// serverConn.(*net.UDPConn).WriteToUDP(buf[0:n], addr)
			_, err = serverConn.Write(buf[0:n])
			if err != nil {
				return
			}
		}
	}()

	buf := make([]byte, 1024)
	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		fmt.Println("Received ", string(buf[0:n]), " from ", addr)

		clientConn, err := net.Dial("udp", addr.String())
		if err != nil {
			return
		}
		defer clientConn.Close()

		_, err = clientConn.Write(buf[0:n])
		if err != nil {
			return
		}
	}
}
