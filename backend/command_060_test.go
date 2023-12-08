package backend

import (
	"testing"

	"connectrpc.com/connect"
	v1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestGetCharactersRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 60, // Command code
		10, 0, // Packet length
		108, 111, 103, 105, 110, 0, // Username = login
	}

	// Act
	req := GetCharactersRequest(packet[4:])
	username, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "login", username)
}

func TestBackend_HandleGetCharacters(t *testing.T) {
	b := &Backend{CharacterClient: &mockCharacterClient{
		ListCharactersResponse: connect.NewResponse(&v1.ListCharactersResponse{
			Characters: []*v1.Character{
				{
					UserId:        1,
					CharacterId:   1,
					CharacterName: "character1",
				},
				{
					UserId:        1,
					CharacterId:   2,
					CharacterName: "character2",
				},
			},
		}),
	}}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: 1, Username: "JP"}

	assert.NoError(t, b.HandleGetCharacters(session, GetCharactersRequest("tester\x00")))
	assert.Equal(t, []byte{255, 60, 34, 0}, conn.Written[0:4])     // Header
	assert.Equal(t, []byte{1, 0, 0, 0}, conn.Written[4:8])         //
	assert.Equal(t, []byte{2, 0, 0, 0}, conn.Written[8:12])        // Number of characters
	assert.Equal(t, []byte("character1\x00"), conn.Written[12:23]) // First character
	assert.Equal(t, []byte("character2\x00"), conn.Written[23:34]) // Second character
	assert.Len(t, conn.Written, 34)
}
