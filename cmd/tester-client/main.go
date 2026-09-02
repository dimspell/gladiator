package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"time"
)

const peerTCP = "127.0.0.2:6114"
const peerUDP = "127.0.0.2:6113"

// firstHelloPacket builds a synthetic hello packet sent to the backend
// under test. The username is a made-up test fixture.
func firstHelloPacket() []byte {
	buf := bytes.NewBuffer([]byte{'#', '#'}) // header
	buf.WriteString("testuser")              // synthetic test username
	buf.WriteByte(0)

	return buf.Bytes()
}

func main() {
	ctx := context.Background()

	// Connect to the TCP peer under test.
	tcpConn, err := net.DialTimeout("tcp", peerTCP, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected TCP", tcpConn.LocalAddr().String(), tcpConn.RemoteAddr().String())
	defer tcpConn.Close()

	udpAddr, err := net.ResolveUDPAddr("udp", peerUDP)
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected UDP", udpConn.LocalAddr().String(), udpConn.RemoteAddr().String())

	go func() {
		time.Sleep(1 * time.Second)

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

	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := tcpConn.Read(buf)
			if err != nil {
				log.Println("READ", err)
				break
			}

			log.Println("TCP", string(buf[:n]))
		}
	}()

	for {
		buf := make([]byte, 1024)
		n, _, err := udpConn.ReadFrom(buf)
		if err != nil {
			log.Println("UDP", err)
			break
		}
		log.Println("UDP", buf[:n])

		if buf[0] == 27 {
			_, _ = udpConn.Write([]byte{13, 0, 2, 0})
		}
		// Note: no synthetic reply for other opcodes here; see packet unit tests.
		if buf[0] == 9 {
			_, _ = udpConn.Write([]byte{53, 0, 2, 0, 0, 0})
			_, _ = udpConn.Write([]byte{2, 39, 2, 0, 0, 39})
		}
	}

	<-ctx.Done()
}
