package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
)

func ListenTCP(i byte) {
	addr := fmt.Sprintf("127.21.37.%d:6114", i)
	fmt.Println(addr)
}

func ListenUDP(i byte) {
	addr := fmt.Sprintf("127.21.37.%d:6113", i)
	fmt.Println(addr)
}

func MockHostTCPServer(ctx context.Context) {
	// Listen for incoming connections.
	tcpListen, err := net.Listen("tcp4", "127.21.37.10:6114")
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	processPackets := func(ctx2 context.Context, conn net.Conn) {
		defer conn.Close()

		destinationConn, err := net.Dial("tcp", "127.0.0.1:6114")
		if err != nil {
			log.Println("Error connecting to destination server:", err)
			return
		}
		defer destinationConn.Close()

		for {
			if ctx2.Err() != nil {
				return
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				log.Printf("error reading: %s\n", err)
				return
			}

			log.Println("TCP", string(buf[:n]), n, buf[:n])

			if _, err := destinationConn.Write(buf[:n]); err != nil {
				log.Println("TCP", "WriteToTCP", err)
			}
		}
	}

	// Close the listener when the application closes.
	defer tcpListen.Close()
	fmt.Println("Listening on", tcpListen.Addr().String())

	for {
		if ctx.Err() != nil {
			return
		}

		// Listen for an incoming connection.
		conn, err := tcpListen.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}
		log.Println("TCP", "Accepted connection")

		go processPackets(ctx, conn)
	}
}

func MockHostUDPServer(ctx context.Context) {
	udpAddr, err := net.ResolveUDPAddr("udp4", "127.21.37.10:6113")
	if err != nil {
		log.Fatal(err)
	}
	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()

	fmt.Println("Listening on", udpAddr.String())

	for {
		if ctx.Err() != nil {
			return
		}

		buffer := make([]byte, 1024)
		n, addr, err := udpConn.ReadFrom(buffer)
		if err != nil {
			break
		}

		// Print out the command sent over UDP
		log.Println("UDP", addr.String(), string(buffer[:n]), buffer[:n])

		// Copy the packet further to main host
		destinationAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:6113")
		if err != nil {
			log.Println("ResolveUDP", err)
		}
		if _, err = udpConn.WriteToUDP(buffer[:n], destinationAddr); err != nil {
			log.Println("WriteToUDP", err)
		}
	}
}

type Proxy struct {
	// Used when the Player is a host and exposes server on tcp:6114
	GameHost any

	// Used when the Player is a guest and communicates with server on tcp:6114
	GameGuest any

	// Used to communicate between players over udp:6113
	PlayerUDP [4]any
}

func (p *Proxy) StartGameHost(ctx context.Context) {
	go p.listenTCP(ctx, "127.21.37.10", "6114")
	go p.listenUDP(ctx, "127.21.37.10", "6113")
}

func (p *Proxy) StartGameGuest(ctx context.Context) {
	go p.listenTCP(ctx, "127.21.37.10", "6114")
	go p.listenUDP(ctx, "127.21.37.10", "6113")
}

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

	fmt.Println("Listening on", udpAddr.String())

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
	fmt.Println("Listening on " + connHost + ":" + connPort)
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
