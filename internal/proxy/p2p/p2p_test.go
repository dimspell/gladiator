package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/proxy/proxytesthelper"
	"github.com/dimspell/gladiator/internal/signalserver"
	"go.uber.org/goleak"
)

const (
	roomName    = "test"
	player1Name = "player1"
	player2Name = "player2"
)

func TestWebRTCMock(t *testing.T) {
	logger.SetJSONLogger(os.Stderr, slog.LevelDebug)

	proxytesthelper.StartHost(t)
	// signalServerURL := proxytesthelper.StartSignalServer(t)

	h, err := signalserver.NewServer()
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h)
	wsURI, _ := url.Parse(ts.URL)
	wsURI.Scheme = "ws"

	signalServerURL := wsURI.String()

	ipRing := NewIpRing()
	ipRing.isTesting = true

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Player 1
	player1, err := DialSignalServer(signalServerURL, "player1", roomName, true)
	if err != nil {
		panic(err)
	}
	player1.IpRing = ipRing
	go player1.Run(ctx)

	// Player 2
	player2, err := DialSignalServer(signalServerURL, "player2", roomName, false)
	if err != nil {
		panic(err)
	}
	player2.IpRing = ipRing
	go player2.Run(ctx)

	// Player 3
	player3, err := DialSignalServer(signalServerURL, "player3", roomName, false)
	if err != nil {
		panic(err)
	}
	player3.IpRing = ipRing
	go player3.Run(ctx)

	<-time.After(3 * time.Second)

	fmt.Println(player1.Peers)
	fmt.Println(player2.Peers)
	fmt.Println(player3.Peers)

	player1.Close()
	player2.Close()
	player3.Close()

	ts.Close()

	t.Cleanup(func() {
		time.Sleep(1 * time.Second)
		goleak.VerifyNone(t)
	})
}
