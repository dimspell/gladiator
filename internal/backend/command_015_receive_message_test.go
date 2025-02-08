package backend

import (
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAppendCharacterToLobby(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, AppendCharacterToLobby("user", model.ClassTypeMage, 0)))
	assert.Equal(t, []byte{
		255, 15, // packet code
		21, 0, // packet length
		2, 0, 0, 0, // op code
		3, 0, 0, 0, // class type = mage
		0, 0, 0, 0, // unused
		'u', 's', 'e', 'r', 0, // user name
	}, conn.Written)
}

func TestRemoveCharacterFromLobby(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, RemoveCharacterFromLobby("user")))
	assert.Equal(t, []byte{
		255, 15, // packet code
		21, 0, // packet length
		3, 0, 0, 0, // op code
		0, 0, 0, 0, // unused
		0, 0, 0, 0, // unused
		'u', 's', 'e', 'r', 0, // user name
	}, conn.Written)
}

func TestNewGlobalMessage(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, NewGlobalMessage("admin", "global message")))
	assert.Equal(t, []byte{
		255, 15, // packet code
		37, 0, // packet length
		4, 0, 0, 0, // op code
		0, 0, 0, 0, // unused
		0, 0, 0, 0, // unused
		'a', 'd', 'm', 'i', 'n', 0, // user name
		'g', 'l', 'o', 'b', 'a', 'l', ' ', 'm', 'e', 's', 's', 'a', 'g', 'e', 0,
	}, conn.Written)
}

func TestNewSystemMessage(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, NewLobbyMessage("user", "lobby message")))
	assert.Equal(t, []byte{
		255, 15, // packet code
		35, 0, // packet length
		5, 0, 0, 0, // op code
		0, 0, 0, 0, // unused
		0, 0, 0, 0, // unused
		'u', 's', 'e', 'r', 0, // user name
		'l', 'o', 'b', 'b', 'y', ' ', 'm', 'e', 's', 's', 'a', 'g', 'e', 0,
		//'u', 'n', 'k', 'n', 'o', 'w', 'n', 0,
	}, conn.Written)
}

func TestSetChannelName(t *testing.T) {
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	assert.NoError(t, session.Send(packet.ReceiveMessage, SetChannelName("DISPEL")))
	assert.Equal(t, []byte{
		255, 15, // packet code
		24, 0, // packet length
		7, 0, 0, 0, // op code
		0, 0, 0, 0, // unused
		0, 0, 0, 0, //unused
		0,                               // unused
		'D', 'I', 'S', 'P', 'E', 'L', 0, // channel name
	}, conn.Written)
}
