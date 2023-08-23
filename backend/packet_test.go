package backend

import (
	"bytes"
	"testing"

	"github.com/dispel-re/dispel-multi/model"
	"github.com/stretchr/testify/assert"
)

func TestClientAuthenticationRequest(t *testing.T) {
	packet := []byte{
		255, 41, // Command code
		19, 0, // Packet length
		2, 0, 0, 0, // Unknown
		112, 97, 115, 115, 0, // Password
		108, 111, 103, 105, 110, 0, // Username
	}
	req := ClientAuthenticationRequest(packet[4:])
	username, password := req.UsernameAndPassword()

	assert.Equal(t, uint32(2), req.Unknown())
	assert.Equal(t, "pass", password)
	assert.Equal(t, "login", username)
}

func TestCreateNewAccountRequest(t *testing.T) {
	packet := []byte{
		255, 42, // Command code
		22, 0, // Packet length
		33, 78, 0, 0, // CD-key
		112, 97, 115, 115, 119, 111, 114, 100, 0, // Password
		117, 115, 101, 114, 0, // User name
		0, 0, 49, 207, 69, 0, // Unknown
	}
	req := CreateNewAccountRequest(packet[4:])
	username, password := req.UsernameAndPassword()

	assert.Equal(t, uint32(20001), req.CDKey())
	assert.Equal(t, "password", password)
	assert.Equal(t, "user", username)
	assert.True(t, bytes.Equal([]byte{0, 0, 49, 207, 69, 0}, req.Unknown()))
}

func TestGetCharactersRequest(t *testing.T) {
	packet := []byte{
		255, 60, // Command code
		10, 0, // Packet length
		108, 111, 103, 105, 110, 0, // Username = login
	}
	req := GetCharactersRequest(packet[4:])

	assert.Equal(t, "login", req.Username())
}

func TestDeleteCharacterRequest_UsernameAndCharacterName(t *testing.T) {
	t.Run("packet parsing", func(t *testing.T) {
		t.Fatal("Move it to the commands testing")

		packet := []byte{
			255, 61, // Command code
			14, 0, // Packet length
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
			0, // Unknown (slot?)
		}
		req := DeleteCharacterRequest(packet[4:])
		user, character := req.UsernameAndCharacterName()

		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("valid names", func(t *testing.T) {
		// Arrange
		data := []byte("user\x00character\x00\x00")
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName := req.UsernameAndCharacterName()

		// Assert
		assert.Equal(t, "user", username)
		assert.Equal(t, "character", characterName)
	})

	t.Run("missing null byte", func(t *testing.T) {
		// Arrange
		data := []byte("usercharacter\x00")
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName := req.UsernameAndCharacterName()

		// Assert
		assert.Empty(t, username)
		assert.Empty(t, characterName)
	})

	t.Run("empty data", func(t *testing.T) {
		// Arrange
		data := []byte{}
		req := DeleteCharacterRequest(data)

		// Act
		username, characterName := req.UsernameAndCharacterName()

		// Assert
		assert.Empty(t, username)
		assert.Empty(t, characterName)
	})
}

func TestCreateCharacterRequest(t *testing.T) {
	packet := []byte{
		255, 92, // Packet header
		65, 0, // Packet length
		20, 0, // Strength
		15, 0, // Agility
		11, 0, // Wisdom
		21, 0, // Constitution
		0, 0, // Health points
		0, 0, // Magic points
		0, 0, 0, 0, // Experience points
		44, 1, 0, 0, // Money
		0, 0, 0, 0, // Score points
		0,    // Class type
		102,  // Skin carnation
		112,  // Hairstyle
		2,    // Slot for light armour (legs)
		7,    // Slot for light armour (torso)
		100,  // Slot for light armour (hands)
		12,   // Slot for light armour (boots)
		100,  // Slot for full armour
		100,  // Slot for emblem
		100,  // Slot for helmet
		100,  // Slot for secondary weapon
		42,   // Slot for primary weapon
		100,  // Slot for shield
		100,  // Unknown slot
		0,    // Gender
		1,    // Character level
		1, 0, // Edged weapons
		2, 0, // Blunted weapons
		1, 0, // Archery
		1, 0, // Polearms
		1, 0, // Wizardry
		0, 0, 0, 0, 0, 0, // Unknown
		117, 115, 101, 114, 0, // User name
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
	}
	req := CreateCharacterRequest(packet)
	user, character := req.UserAndCharacterName()
	info := req.CharacterInfo()

	assert.True(t, bytes.Equal(packet[4:60], info.ToBytes()))
	assert.Equal(t, "user", user)
	assert.Equal(t, "character", character)
}

func TestSelectChannelRequest(t *testing.T) {
	packet := []byte{
		255, 12,
		0, 0,
		99, 104, 97, 110, 110, 101, 108, 0,
		68, 73, 83, 80, 69, 76, 0,
	}
	req := SelectChannelRequest(packet[4:])

	assert.Equal(t, "channel", req.ChannelName())
}

func TestRankingRequest(t *testing.T) {
	t.Run("NewRankingRequest", func(t *testing.T) {
		req := NewRankingRequest(model.ClassTypeKnight, 1000, "user", "character")
		user, character := req.UserAndCharacterName()

		assert.Equal(t, model.ClassTypeKnight, req.ClassType())
		assert.Equal(t, uint32(1000), req.Offset())
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("First page for warrior", func(t *testing.T) {
		packet := []byte{
			255, 70, // Command code
			27, 0, // Packet length
			1, 0, 0, 0, // Class type
			0, 0, 0, 0, // Offset
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		}
		req := RankingRequest(packet[4:])
		user, character := req.UserAndCharacterName()

		assert.Equal(t, model.ClassTypeWarrior, req.ClassType())
		assert.Equal(t, uint32(0), req.Offset())
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})

	t.Run("Second page for mage", func(t *testing.T) {
		packet := []byte{
			255, 70, // Command code
			22, 0, // Packet length
			3, 0, 0, 0, // Class type
			10, 0, 0, 0, // Offset
			117, 115, 101, 114, 0, // User name
			99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		}
		req := RankingRequest(packet[4:])
		user, character := req.UserAndCharacterName()

		assert.Equal(t, model.ClassTypeMage, req.ClassType())
		assert.Equal(t, uint32(10), req.Offset())
		assert.Equal(t, "user", user)
		assert.Equal(t, "character", character)
	})
}
