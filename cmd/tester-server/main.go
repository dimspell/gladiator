package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

const backendIP = "192.168.121.212"

func main() {
	ctx := context.TODO()

	p := Proxy{}
	go p.listenTCP(ctx, backendIP, "6114")
	go p.listenUDP(ctx, backendIP, "6113")

	fmt.Println("Waiting...")
	<-ctx.Done()
}

type Proxy struct{}

func (p *Proxy) listenUDP(ctx context.Context, connHost, connPort string) {
	udpAddr, err := net.ResolveUDPAddr("udp", connHost+":"+connPort)
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
		fmt.Println("udp - waiting for read")
		if ctx.Err() != nil {
			return
		}

		buf := make([]byte, 1024)
		n, addr, err := udpConn.ReadFrom(buf)
		if err != nil {
			break
		}

		if buf[0] == 26 {
			udpConn.Write([]byte{26, 0, 2, 0})
		}

		fmt.Println(connPort, addr.String(), string(buf[:n]), buf[:n])
	}
}

func (p *Proxy) listenTCP(ctx context.Context, connHost, connPort string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	processPackets := func(conn net.Conn) {
		defer conn.Close()

		for {
			fmt.Println("waiting for read")

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Printf("error reading (%s): %s\n", connPort, err)
				return
			}
			fmt.Println(connPort, string(buf[:n]), n, buf[:n])

			conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
			// conn.Write(buf[:n])
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
	fmt.Println("DONE")
}
