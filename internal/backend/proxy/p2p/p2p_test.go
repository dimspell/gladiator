package p2p

import (
	"log/slog"
	"testing"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.SetDiscardLogger()
}

func TestPeerID(t *testing.T) {
	assert.Equal(t, "123", peerID(123))
	assert.Equal(t, "0", peerID(0))
}

func TestParseUserID(t *testing.T) {
	id, err := parseUserID("123")
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)

	id, err = parseUserID("0")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), id)
}

func TestPeer_Close(t *testing.T) {
	peer := &Peer{
		peerID: "1",
		logger: slog.Default(),
	}
	// Should not panic even with nil connection/datachannel
	peer.Close()
}
