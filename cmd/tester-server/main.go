package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

const backendIP = "127.0.1.28"

func main() {
	ctx := context.TODO()

	p := Proxy{}
	go p.listenTCP(ctx, "127.0.0.1", "6114")
	go p.listenUDP(ctx, "127.0.0.1", "6113")

	fmt.Println("Waiting...")
	<-ctx.Done()
}

type Proxy struct{}

func (p *Proxy) listenUDP(ctx context.Context, connHost, connPort string) {
	udpAddr, err := net.ResolveUDPAddr("udp", connHost+":"+connPort)
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()

	log.Println("Listening UDP on", udpAddr.String())

	for {
		if ctx.Err() != nil {
			fmt.Println("context err")
			return
		}

		buf := make([]byte, 1024)
		n, addr, err := udpConn.ReadFrom(buf)
		if err != nil {
			break
		}

		fmt.Println("Accepted UDP connection", addr.String())

		fmt.Println(connPort, addr.String(), string(buf[:n]), buf[:n])

		if buf[0] == 26 {
			{
				_, err = udpConn.WriteToUDP([]byte{27, 0, 2, 0}, udpAddr)
				fmt.Println(err)
			}
			fmt.Println("Responded with 27")
		}
	}
}

func (p *Proxy) listenTCP(ctx context.Context, connHost, connPort string) {
	// Listen for incoming connections.
	l, err := net.Listen("tcp", connHost+":"+connPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	log.Println("Listening TCP on", l.Addr().String())

	processPackets := func(conn net.Conn) {
		defer conn.Close()

		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Printf("error reading (%s): %s\n", connPort, err)
				return
			}
			log.Println(connPort, string(buf[:n]), n, buf[:n])

			if _, err := conn.Write([]byte{35, 35, 116, 101, 115, 116, 0}); err != nil {
				log.Println(err)
			}
			// conn.Write(buf[:n])
		}
	}

	// Close the listener when the application closes.
	defer l.Close()
	for {
		if ctx.Err() != nil {
			return
		}

		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}
		log.Println("Accepted TCP connection", conn.RemoteAddr().String())

		go processPackets(conn)
	}
}
