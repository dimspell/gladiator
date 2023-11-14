package backend

import (
	"context"
	"database/sql"
	"encoding/base64"
	"testing"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestGetCharacterSpells(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 72, // Command code
		19, 0, // Packet length
		117, 115, 101, 114, 0, // User name
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
	}

	// Act
	req := GetCharacterSpellsRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "user", data.Username)
	assert.Equal(t, "character", data.CharacterName)
}

func TestBackend_HandleGetCharacterSpells(t *testing.T) {
	db := testDB(t)
	user, err := db.CreateUser(context.TODO(), database.CreateUserParams{
		Username: "tester",
		Password: "password",
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	db.CreateCharacter(context.TODO(), database.CreateCharacterParams{
		CharacterName: "characterName",
		UserID:        user.ID,
		SortOrder:     1,
	})

	spells := make([]byte, 43)
	for i := 0; i < 41; i++ {
		spells[i] = 1
	}
	spells[0] = 2

	db.UpdateCharacterSpells(context.TODO(), database.UpdateCharacterSpellsParams{
		Spells: sql.NullString{
			Valid:  true,
			String: base64.StdEncoding.EncodeToString(spells),
		},
		CharacterName: "characterName",
		UserID:        user.ID,
	})

	b := &Backend{DB: db}
	conn := &mockConn{}
	session := &model.Session{ID: "TEST", Conn: conn, UserID: user.ID, Username: "JP"}

	assert.NoError(t, b.HandleGetCharacterSpells(session, GetCharacterSpellsRequest("tester\x00characterName\x00")))
	assert.Equal(t, []byte{255, 72, 47, 0}, conn.Written[0:4]) // Header
	assert.Equal(t, spells, conn.Written[4:47])                // Spells
	assert.Len(t, conn.Written, 47)
}
