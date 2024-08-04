package proxy

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/lmittmann/tint"
)

func TestPeerToPeer(t *testing.T) {
	// slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
	slog.SetDefault(slog.New(
		tint.NewHandler(
			os.Stderr,
			&tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
			},
		),
	))

	// t.Run("Tester helpers", func(t *testing.T) {
	// 	StartHost(t)
	// 	StartSignalServer(t)
	//
	// 	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", "6114"), 3*time.Second)
	// 	if err != nil {
	// 		t.Error(err)
	// 		return
	// 	}
	//
	// 	if _, err := conn.Write([]byte("hello")); err != nil {
	// 		t.Error(err)
	// 		return
	// 	}
	// })

	t.Run("Hosting a game", func(t *testing.T) {
		const roomName = "room"

		StartHost(t)
		websocketURL := StartSignalServer(t)

		a := NewPeerToPeer(websocketURL)
		if err := a.dialSignalServer("user1", roomName); err != nil {
			t.Error(err)
			return
		}
		if err := a.Host(GameRoom(roomName), User("user1")); err != nil {
			t.Error(err)
			return
		}

		b := NewPeerToPeer(websocketURL)
		if err := b.dialSignalServer("user2", roomName); err != nil {
			t.Error(err)
			return
		}
		if _, err := b.Join(roomName, "user", "user2", ""); err != nil {
			t.Error(err)
			return
		}

		time.Sleep(2 * time.Second)
	})

	// t.Run("Joining a game", func(t *testing.T) {
	//
	// })

	// t.Run("Switching host", func(t *testing.T) {
	//
	// })
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
					fmt.Println(err)
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

func StartSignalServer(t testing.TB) string {
	t.Helper()

	s, err := signalserver.NewServer()
	if err != nil {
		t.Fatal(err)
	}

	start, stop := s.Run()

	go func() {
		start(context.TODO())
	}()

	t.Cleanup(func() {
		stop(context.TODO())
	})

	const websocketURL = "ws://localhost:5050"
	return websocketURL
}
