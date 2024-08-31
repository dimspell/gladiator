package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dimspell/gladiator/console/signalserver"
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

	msgChan chan []byte
	closed  bool

	onClose   func()
	onMessage func(msg webrtc.DataChannelMessage)
}

func newFakeDataChannel(label string) *FakeDataChannel {
	return &FakeDataChannel{
		label:  label,
		Buffer: [][]byte{},
	}
}

func (f *FakeDataChannel) Label() string { return f.label }

func (f *FakeDataChannel) OnError(fn func(err error)) {
	// TODO implement me
}

func (f *FakeDataChannel) OnMessage(fn func(msg webrtc.DataChannelMessage)) {
	f.onMessage = fn
}

func (f *FakeDataChannel) OnClose(fn func()) {
	f.onClose = fn
}

func (f *FakeDataChannel) Send(p []byte) error {
	f.Buffer = append(f.Buffer, p)
	return nil
}

func (f *FakeDataChannel) Close() error {
	if f.closed {
		return fmt.Errorf("already closed")
	}
	f.onClose()
	return nil
}

func StartSignalServer(t testing.TB) string {
	t.Helper()

	h, err := signalserver.NewServer()
	if err != nil {
		t.Fatal(err)
		return ""
	}
	ts := httptest.NewServer(h)

	t.Cleanup(func() {
		ts.Close()
	})

	wsURI, _ := url.Parse(ts.URL)
	wsURI.Scheme = "ws"

	return wsURI.String()
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
