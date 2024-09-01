package proxytesthelper

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"
)

func StartHost(t testing.TB) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.TODO())

	// Listen for incoming connections.
	tcpListener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "6114"))
	if err != nil {
		log.Fatal(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", "6113"))
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// Listen UDP
	go func() {
		for {
			if ctx.Err() != nil {
				fmt.Println("context err")
				return
			}

			buf := make([]byte, 1024)
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				break
			}

			if buf[0] == '#' {
				resp := append([]byte{27, 0}, buf[1:n]...)
				_, err := udpConn.WriteToUDP(resp, udpAddr)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println("UDP response", string(resp))
			}
		}
	}()

	processPackets := func(conn net.Conn) {
		message := make(chan []byte, 1)

		go func() {
			defer conn.Close()

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-message:
					if !ok {
						return
					}
					log.Println("Message received", string(msg))
					conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
				}
			}
		}()

		for {
			conn.SetDeadline(time.Now().Add(10 * time.Second))

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				close(message)
				return
			}
			message <- buf[:n]
		}
	}

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			// Listen for an incoming connection.
			conn, err := tcpListener.Accept()
			if err != nil {
				continue
			}
			go processPackets(conn)
		}
	}()

	t.Cleanup(func() {
		cancel()
		udpConn.Close()
		tcpListener.Close()
	})
}
