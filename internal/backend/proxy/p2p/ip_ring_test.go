package p2p

import (
	"testing"

	"github.com/dimspell/gladiator/console/signalserver"
)

func TestIpRing_CreateClient(t *testing.T) {
	const (
		roomName    = "test"
		player1Name = "player1"
		player2Name = "player2"
		player3Name = "player3"
	)

	t.Run("I am a host, one is joining me", func(t *testing.T) {
		StartHost(t)

		r := NewIpRing()
		r.isTesting = true

		// Hosting player on localhost tcp:6114, udp:6113
		other := signalserver.Member{
			UserID: player2Name,
			IsHost: false,
			Joined: false,
		}
		tcpProxy, udpProxy, err := r.CreateClient(true, other)
		if err != nil {
			t.Error(err)
			return
		}
		defer tcpProxy.Close()
		defer udpProxy.Close()

		t.Log(tcpProxy, udpProxy)
	})

	t.Run("I am a guest, joining to the host", func(t *testing.T) {
		r := NewIpRing()
		r.isTesting = true

		// Hosting player on localhost tcp:6114, udp:6113
		other := signalserver.Member{
			UserID: player2Name,
			IsHost: true,
			Joined: true,
		}
		tcpProxy, udpProxy, err := r.CreateClient(false, other)
		if err != nil {
			t.Error(err)
			return
		}
		defer tcpProxy.Close()
		defer udpProxy.Close()

		t.Log(tcpProxy, udpProxy)
	})
}
