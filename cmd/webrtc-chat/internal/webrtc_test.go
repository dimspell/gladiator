package internal

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

func TestWebRTC(t *testing.T) {
	signalingURL := StartSignalServer(t)

	// Player 1
	{
		player1, err := Dial(&DialParams{
			SignalingURL: signalingURL,
			RoomName:     "test",
			ID:           uuid.New().String()[:6],
		})
		if err != nil {
			panic(err)
		}
		go player1.Run(func(peer *Peer, packet webrtc.DataChannelMessage) {
			fmt.Println("Received:", string(packet.Data))
		})
	}

	// Player 2
	{
		player2, err := Dial(&DialParams{
			SignalingURL: signalingURL,
			RoomName:     "test",
			ID:           uuid.New().String()[:6],
		})
		if err != nil {
			panic(err)
		}
		go player2.Run(func(peer *Peer, packet webrtc.DataChannelMessage) {
			fmt.Println("Received:", string(packet.Data))
		})
	}

	<-time.After(3 * time.Second)
}

func TestWebRTCMock(t *testing.T) {

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
