package p2p

import (
	"fmt"
	"log"
	"testing"

	"github.com/dimspell/gladiator/internal/proxy/proxytesthelper"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
)

func TestNewPipe(t *testing.T) {
	t.Run("I am a host, one is joining me", func(t *testing.T) {
		proxytesthelper.StartHost(t)

		r := NewIpRing()
		r.isTesting = true

		dc := proxytesthelper.NewFakeDataChannel(fmt.Sprint(roomName, "/udp"))

		peer1 := r.NextPeerAddress(player1Name, true, true)
		_, proxy1, err := redirect.New(peer1.Mode, peer1.Addr)
		if err != nil {
			t.Error(err)
			return
		}

		player2 := NewPipe(dc, proxy1)
		defer player2.Close()

		peer2 := r.NextPeerAddress(player2Name, false, false)
		_, proxy2, err := redirect.New(peer2.Mode, peer2.Addr)
		if err != nil {
			t.Error(err)
			return
		}

		player1 := NewPipe(dc, proxy2)
		defer player1.Close()

		if _, err := player2.Write([]byte("#hello")); err != nil {
			t.Error(err)
			return
		}

		log.Println(dc.Buffer)
	})
}
