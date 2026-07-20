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

// gameServerIP defines IP address of the game server (DispelMulti.exe process)
// const gameServerIP = "192.168.121.212"

// clientIP defines IP address of the client (this application)
// const clientIP = "192.168.121.212"

// [26 0 2 0]
// [27 0 2 0]

func firstHelloPacket() []byte {
	// buf := bytes.NewBuffer([]byte{35, 35}) // header
	buf := bytes.NewBuffer([]byte{'#', '#'}) // header
	buf.WriteString("admin")                 // username (used in login)
	buf.WriteByte(0)

	return buf.Bytes()
}

func main() {
	ctx := context.TODO()

	// p := Proxy{}
	// p.listenUDP(ctx, clientIP, "6113")

	// Connect to tcp:6114 over TCP
	// tcpConn, err := net.Dial("tcp4", "127.21.37.10:6114")
	// tcpConn, err := net.Dial("tcp4", "127.0.0.1:6114")
	// tcpConn, err := net.Dial("tcp", fmt.Sprintf("%s:6114", gameServerIP))
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
		if buf[0] == 14 {
			_, _ = udpConn.Write([]byte{8, 220, 2, 0, 1, 102, 113, 0, 255, 15, 14, 255, 255, 255, 255, 255, 73, 'm', 'a', 'g', 'e', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'm', 'a', 'g', 'e', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		}
		if buf[0] == 9 {
			_, _ = udpConn.Write([]byte{53, 0, 2, 0, 0, 0})
			_, _ = udpConn.Write([]byte{2, 39, 2, 0, 0, 39})
		}
	}

	// => 26 0 2 0
	// <= 27 0 2 0
	// => 13 0 2 0
	// <= [14 0 2 0 0 0 0 0 0 102 113 0 0 0 0 0 7 2 12 255 255 255 113 255 66 97 114 99 104 101 114 0 0 0 0 0 0 0 0 0 0 0 0 0] archer
	// => [8 220 2 0 1 102 113 0 255 15 14 255 255 255 255 255 73 109 97 103 101 0 0 0 0 0 0 0 0 0 0 0 109 97 103 101 0 0 0 0 0 0 0 0 0 0 0 0] mage (109 97 103 101
	// => 9 0 2 0
	// <= 53 28 2 0 0 28
	// <= 2 39 2 0 0 39

	// Tested:

	// Move comamand

	// UDP [1 198 2 0 1 0 0 0 188 9 0 0 189 9 0 0 104 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 189 9 0 0 190 9 0 0 105 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 190 9 0 0 191 9 0 0 106 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 191 9 0 0 192 9 0 0 107 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 192 9 0 0 193 9 0 0 108 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 193 9 0 0 194 9 0 0 109 5 0 0 0 0 63 0 33 0 0 0]
	// UDP [1 198 2 0 1 0 0 0 194 9 0 0 195 9 0 0 110 5 0 0 0 0 63 0 33 0 0 0]
	//                        194 = coords
	//                                  195 coords

	// UDP [1 198 2 0 7 0 0 0 41 13 0 0 223 12 0 0 213 8 0 0 0 0 63 0 33 0 0 0]
	//      1 198 = Move Command
	//            2 0 = Player ID
	//                7 0 = (value between 0 and 8) - the rotation of the move

	// Stop moving
	// UDP [15 0 2 0 7 0 0 0 113 9 0 0 0 0 0 0 172 26 32 1]

	// Auth command
	// => 26 0 2 0
	// <= 27 0 2 0
	// => 13 0 2 0
	// <= DgACAAAAAAAAZnEAAAAAAAcCDP///3H/QmFyY2hlcgAAAAAAAAAAAAAAAAA= // 14 0 2 0 ...
	// => CNwCAAFmcQD/Dw7//////0ltYWdlAAAAAAAAAAAAAABtYWdlAAAAAAAAAAAAAAAA // 8 220 2 0 1 ...
	// => 9 0 2 0
	// <= 53 28 2 0 0 28
	// <= 2 39 2 0 0 39

	// Change equipment (take off sword)
	// UDP [6 0 2 0 0 7 2 12 255 255 255 112 255 255 0 0 172 26 32 1]
	// UDP [6 0 2 0 0 7 2 12 255 255 255 112 255 255 71 0 172 26 32 1]

	// Put sword back
	// UDP [6 0 2 0 0 7 2 12 255 255 255 112 255 42 71 0 172 26 32 1]
	// UDP [6 252 2 0 0 7 2 12 255 255 255 112 255 42 0 0 172 26 32 1]

	// Take off the armor on torso
	// UDP [6 8 2 0 0 7 255 12 255 255 255 112 255 42 0 0 172 26 32 1]
	// UDP [6 0 2 0 0 7 255 12 255 255 255 112 255 42 71 0 172 26 32 1]

	// Put armor back
	// UDP [6 0 2 0 0 7 2 12 255 255 255 112 255 42 71 0 172 26 32 1]
	// UDP [6 252 2 0 0 7 2 12 255 255 255 112 255 42 0 0 172 26 32 1]

	// Take off the trousers
	// UDP [6 10 2 0 0 255 2 12 255 255 255 112 255 42 0 0 172 26 32 1]
	// UDP [6 0 2 0 0 255 2 12 255 255 255 112 255 42 71 0 172 26 32 1]

	// Put them back
	// UDP [6 0 2 0 0 7 2 12 255 255 255 112 255 42 71 0 172 26 32 1]
	// UDP [6 252 2 0 0 7 2 12 255 255 255 112 255 42 0 0 172 26 32 1]

	// Replace trousers with hide trousers
	// UDP [6 0 2 0 0 6 2 12 255 255 255 112 255 42 71 0 172 26 32 1]
	// UDP [6 252 2 0 0 6 2 12 255 255 255 112 255 42 0 0 172 26 32 1]

	// Attempt to cast magic (but blocked in city)
	// UDP [25 0 2 0 223 8 0 0 20 0 0 0 0 0 67 0 172 26 32 1]

	// Same but other spell (the divine armour)
	// UDP [25 0 2 0 40 9 0 0 16 0 0 0 3 0 67 0 172 26 32 1]
	// UDP [25 0 2 0 39 9 0 0 16 0 0 0 5 0 67 0 172 26 32 1]
	// UDP [25 0 2 0 37 9 0 0 16 0 0 0 4 0 67 0 172 26 32 1]

	// Throw money
	// UDP [30 86 2 0 0 252 25 0 44 1 0 0 110 9 0 0 1 0 4 0 0 0 0 0]

	// Take money back
	// UDP [32 0 2 0 0 0 0 0 0 0 0 0 110 9 0 0 0 0 4 0 1 0 0 0]

	<-ctx.Done()
}

// type Proxy struct{}

// func (p *Proxy) listenUDP(ctx context.Context, connHost, connPort string) {
// 	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:6113", clientIP))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	go func() {
// 		udpListener, err := net.ListenUDP("udp", udpAddr)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		defer udpListener.Close()
//
// 		buf := make([]byte, 1024)
// 		log.Println("Listening on", udpListener.LocalAddr().String())
// 		for {
// 			n, addr, err := udpListener.ReadFromUDP(buf)
// 			if err != nil {
// 				log.Fatal(err)
// 			}
//
// 			// handle UDP packet
// 			log.Printf("Received %d bytes from %s", n, addr.String())
// 			log.Println(buf[:n])
// 		}
// 	}()
// }
//
// func (p *Proxy) listenTCP(ctx context.Context, connHost, connPort string) {
// 	// Listen for incoming connections.
// 	l, err := net.Listen("tcp4", connHost+":"+connPort)
// 	if err != nil {
// 		log.Println("Error listening:", err.Error())
// 		os.Exit(1)
// 	}
//
// 	processPackets := func(conn net.Conn) {
// 		defer conn.Close()
//
// 		for {
// 			buf := make([]byte, 1024)
// 			n, err := conn.Read(buf)
// 			if err != nil {
// 				log.Printf("error reading (%s): %s\n", connPort, err)
// 				return
// 			}
// 			log.Println(connPort, string(buf[:n]), n, buf[:n])
// 		}
// 	}
//
// 	// Close the listener when the application closes.
// 	defer l.Close()
// 	log.Println("Listening TCP on", l.Addr().String())
// 	for {
// 		if ctx.Err() != nil {
// 			return
// 		}
//
// 		// Listen for an incoming connection.
// 		conn, err := l.Accept()
// 		if err != nil {
// 			log.Println("Error accepting: ", err.Error())
// 			continue
// 		}
// 		log.Println("Accepted connection on port", connPort)
//
// 		go processPackets(conn)
// 	}
// }
