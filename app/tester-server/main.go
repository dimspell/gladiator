package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	ctx := context.TODO()

	p := Proxy{}
	go p.listenTCP(ctx, "127.0.0.28", "6114")
	go p.listenUDP(ctx, "127.0.0.28", "6113")

	<-ctx.Done()
}

type Proxy struct{}

func (p *Proxy) listenUDP(ctx context.Context, connHost, connPort string) {
	udpAddr, err := net.ResolveUDPAddr("udp4", connHost+":"+connPort)
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()

	fmt.Println("Listening UDP on", udpAddr.String())

	for {
		if ctx.Err() != nil {
			return
		}

		buf := make([]byte, 1024)
		n, addr, err := udpConn.ReadFrom(buf)
		if err != nil {
			break
		}

		log.Fatalln(connPort, addr.String(), string(buf[:n]), buf[:n])
	}
}

func (p *Proxy) listenTCP(ctx context.Context, connHost, connPort string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp4", connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	processPackets := func(conn net.Conn) {
		defer conn.Close()

		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Printf("error reading (%s): %s\n", connPort, err)
				return
			}
			fmt.Println(connPort, string(buf[:n]), n, buf[:n])
		}
	}

	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening TCP on", l.Addr().String())
	for {
		if ctx.Err() != nil {
			return
		}

		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			continue
		}
		fmt.Println("Accepted connection on port", connPort)

		go processPackets(conn)
	}
}
