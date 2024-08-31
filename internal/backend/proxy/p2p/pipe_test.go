package p2p

import (
	"fmt"
	"log"
	"testing"

	"github.com/dimspell/gladiator/console/signalserver"
)

func TestNewPipe(t *testing.T) {
	t.Run("I am a host, one is joining me", func(t *testing.T) {
		StartHost(t)

		r := NewIpRing()
		r.isTesting = true

		dc := newFakeDataChannel(fmt.Sprint(roomName, "/udp"))

		tcpProxyHost, _, err := r.CreateClient(false, signalserver.Member{
			UserID: player2Name,
			IsHost: false,
			Joined: false,
		})
		if err != nil {
			t.Error(err)
			return
		}
		// defer tcpProxyHost.Close()
		// defer udpProxyHost.Close()

		player2 := NewPipe(dc, tcpProxyHost)
		defer player2.Close()

		tcpProxyGuest2, _, err := r.CreateClient(true, signalserver.Member{
			UserID: player2Name,
			IsHost: false,
			Joined: false,
		})
		if err != nil {
			t.Error(err)
			return
		}
		// defer tcpProxyGuest2.Close()
		// defer udpProxyGuest2.Close()

		player1 := NewPipe(dc, tcpProxyGuest2)
		defer player1.Close()

		if _, err := player2.Write([]byte("#hello")); err != nil {
			t.Error(err)
			return
		}

		log.Println(dc.Buffer)
	})
}
