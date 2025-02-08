package backend

import (
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetChannelName(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, SetChannelName("DISPEL")))

	assert.Equal(t, []byte{
		255, 15,
		24, 0,
		7, 0, 0, 0, // 8
		0, 0, 0, 0, // 12
		0, 0, 0, 0, // 16
		0,                            // 17
		'D', 'I', 'S', 'P', 'E', 'L', // 23
		0, // 24
	}, conn.Written) // Header
	assert.Len(t, conn.Written, 24)
}
