package p2p

import (
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/lmittmann/tint"
	"go.uber.org/goleak"
)

func TestWebRTCMock(t *testing.T) {
	defer goleak.VerifyNone(t)

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

	StartHost(t)
	signalServerURL := StartSignalServer(t)

	ipRing := NewIpRing()
	ipRing.isTesting = true

	// Player 1
	player1, err := DialSignalServer(signalServerURL, "player1", roomName, true)
	if err != nil {
		panic(err)
	}
	player1.IpRing = ipRing
	go player1.Run("player1")

	// Player 2
	player2, err := DialSignalServer(signalServerURL, "player2", roomName, false)
	if err != nil {
		panic(err)
	}
	player2.IpRing = ipRing
	go player2.Run("player1")

	// Player 3
	player3, err := DialSignalServer(signalServerURL, "player3", roomName, false)
	if err != nil {
		panic(err)
	}
	player3.IpRing = ipRing
	go player3.Run("player1")

	<-time.After(3 * time.Second)

	fmt.Println(player1.Peers)
	fmt.Println(player2.Peers)
	fmt.Println(player3.Peers)

	player1.Close()
	player2.Close()
	player3.Close()
}
