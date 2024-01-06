package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func firstHelloPacket() []byte {
	buf := bytes.NewBuffer([]byte{35, 35}) // header
	buf.WriteString("hoster")              // game room name
	buf.WriteByte(0)

	return buf.Bytes()
}

func main() {
	ctx := context.TODO()

	p := Proxy{}
	go p.listenUDP(ctx, "127.0.0.34", "6113")

	// Connect to tcp:6114 over TCP
	// tcpConn, err := net.Dial("tcp4", "127.21.37.10:6114")
	tcpConn, err := net.Dial("tcp4", "127.0.0.1:6114")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected", tcpConn.LocalAddr().String(), tcpConn.RemoteAddr().String())
	defer tcpConn.Close()

	udpConn, err := net.Dial("udp4", "127.0.0.1:6113")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected UDP", udpConn.LocalAddr().String(), udpConn.RemoteAddr().String())

	go func() {
		time.Sleep(1 * time.Second)
		log.Println("Writing")
		var n int
		var err error

		n, err = udpConn.Write([]byte{26, 0, 2, 0})
		if err != nil {
			log.Println("WRITE", err)
		}
		log.Println("Wrote", []byte{26, 0, 2, 0}[:n])

		// Write to tcp:6114 over TCP
		log.Println("payload", firstHelloPacket())
		n, err = tcpConn.Write(firstHelloPacket())
		if err != nil {
			log.Println("WRITE", err)
		}
		log.Println("Wrote", firstHelloPacket()[:n])

	}()

	for {
		log.Println("waiting for read")

		buf := make([]byte, 1024)
		n, err := tcpConn.Read(buf)
		if err != nil {
			log.Println("READ", err)
			break
		}

		fmt.Println(string(buf[:n]))
	}

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
