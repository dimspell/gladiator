package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

var _ WebSocket = (*FakeWebsocket)(nil)

type FakeWebsocket struct {
	Buffer [][]byte
	i      int
}

func (f *FakeWebsocket) Close() error {
	// TODO implement me
	panic("implement me")
}

func (f *FakeWebsocket) Read(p []byte) (n int, err error) {
	log.Println("FakeWebsocket.Read", f.i)
	f.i++
	if f.i > len(f.Buffer) {
		return 0, fmt.Errorf("no more messages")
	}
	return copy(p, f.Buffer[f.i-1]), nil
}

func (f *FakeWebsocket) Write(p []byte) (n int, err error) {
	log.Println("FakeWebsocket.Write", len(f.Buffer))
	f.Buffer = append(f.Buffer, p)
	return len(p), nil
}

var _ DataChannel = (*FakeDataChannel)(nil)

type FakeDataChannel struct {
	label string

	Buffer [][]byte
	i      int
}

func (f *FakeDataChannel) Label() string { return f.label }

func (f *FakeDataChannel) OnError(fn func(err error)) {
	// TODO implement me
	panic("implement me")
}

func (f *FakeDataChannel) OnMessage(fn func(msg webrtc.DataChannelMessage)) {
	// TODO implement me
	panic("implement me")
}

func (f *FakeDataChannel) OnClose(fn func()) {
	// TODO implement me
	panic("implement me")
}

func (f *FakeDataChannel) Send(bytes []byte) error {
	// TODO implement me
	panic("implement me")
}

func (f *FakeDataChannel) Close() error {
	// TODO implement me
	panic("implement me")
}

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
			_, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				break
			}

			if buf[0] == 26 {
				{
					_, err = udpConn.WriteToUDP([]byte{27, 0, 2, 0}, udpAddr)
					log.Println(err)
				}
				fmt.Println("Responded with 27")
			}
		}
	}()

	go func() {
		processPackets := func(conn net.Conn) {
			defer conn.Close()

			for {
				conn.SetDeadline(time.Now().Add(10 * time.Second))

				buf := make([]byte, 1024)
				if _, err := conn.Read(buf); err != nil {
					return
				}

				conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
			}
		}

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
