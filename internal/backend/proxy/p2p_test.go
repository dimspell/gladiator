package proxy

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/lmittmann/tint"
	"go.uber.org/goleak"
)

func TestPeerToPeer(t *testing.T) {
	defer goleak.VerifyNone(t)

	// slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true})))
	slog.SetDefault(slog.New(
		tint.NewHandler(
			os.Stderr,
			&tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
				AddSource:  true,
			},
		),
	))

	t.Run("Hosting a game", func(t *testing.T) {
		const roomName = "room"

		// StartHost(t)
		websocketURL := StartSignalServer(t)
		// websocketURL := "ws://localhost:5050"

		a := NewPeerToPeer(websocketURL)
		defer a.Close()

		if _, err := a.Create(CreateParams{
			HostUserIP: "",
			HostUserID: "user1",
			GameID:     roomName,
		}); err != nil {
			t.Error(err)
			return
		}
		if err := a.Host(HostParams{
			GameID:     roomName,
			HostUserID: "user1",
		}); err != nil {
			t.Error(err)
			return
		}

		b := NewPeerToPeer(websocketURL)
		defer b.Close()

		if _, err := b.Join(JoinParams{
			HostUserID:    "user1",
			GameID:        roomName,
			CurrentUserIP: "",
			CurrentUserID: "user2",
		}); err != nil {
			t.Error(err)
			return
		}

		time.Sleep(2 * time.Second)

		fmt.Println(a.Peers)
		fmt.Println(b.Peers)

		if _, err := b.Exchange(ExchangeParams{
			GameID:    roomName,
			UserID:    "user1",
			IPAddress: "127.0.0.1",
		}); err != nil {
			t.Error(err)
			return
		}
	})
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
