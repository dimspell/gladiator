package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

const gameServerIP = "192.168.121.169"
const clientIP = "192.168.121.212"

func firstHelloPacket() []byte {
	// buf := bytes.NewBuffer([]byte{35, 35}) // header
	buf := bytes.NewBuffer([]byte{'#', '#'}) // header
	buf.WriteString("mage")                  // user name (used in login)
	buf.WriteByte(0)

	return buf.Bytes()
}

func main() {
	ctx := context.TODO()

	p := Proxy{}
	p.listenUDP(ctx, clientIP, "6113")

	// Connect to tcp:6114 over TCP
	// tcpConn, err := net.Dial("tcp4", "127.21.37.10:6114")
	// tcpConn, err := net.Dial("tcp4", "127.0.0.1:6114")
	tcpConn, err := net.Dial("tcp", fmt.Sprintf("%s:6114", gameServerIP))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected TCP", tcpConn.LocalAddr().String(), tcpConn.RemoteAddr().String())
	defer tcpConn.Close()

	udpConn, err := net.Dial("udp", fmt.Sprintf("%s:6113", gameServerIP))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected UDP", udpConn.LocalAddr().String(), udpConn.RemoteAddr().String())

	go func() {
		log.Println("Writing")
		var n int
		var err error

		{
			udpPacket := []byte{26, 0, 2, 0}

			n, err = udpConn.Write(udpPacket)
			if err != nil {
				log.Println("WRITE", err)
			}
			log.Println("Wrote UDP", udpPacket[:n])
		}

		// Write to tcp:6114 over TCP
		log.Println("Payload to write", firstHelloPacket())
		n, err = tcpConn.Write(firstHelloPacket())
		if err != nil {
			log.Println("WRITE", err)
		}
		log.Println("Wrote TCP", firstHelloPacket()[:n])
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
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:6113", clientIP))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		udpListener, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			log.Fatal(err)
		}
		defer udpListener.Close()

		buf := make([]byte, 1024)
		log.Println("Listening on", udpListener.LocalAddr().String())
		for {
			n, addr, err := udpListener.ReadFromUDP(buf)
			if err != nil {
				log.Fatal(err)
			}

			// handle UDP packet
			log.Printf("Received %d bytes from %s", n, addr.String())
			fmt.Println(buf[:n])
		}
	}()
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
