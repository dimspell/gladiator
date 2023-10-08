package backend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterStatsRequest(t *testing.T) {
	// Arrange
	packet := []byte{
		255, 108, // Packet header
		76, 0, // Packet length
		100, 0, // Strength
		100, 0, // Agility
		100, 0, // Wisdom
		100, 0, // Constitution
		44, 1, // Health points
		49, 1, // Magic points
		0, 0, 0, 0, // Experience points
		32, 161, 7, 0, // Money
		0, 0, 0, 0, // Score points
		3,    // Class type
		102,  // Skin carnation
		113,  // Hairstyle
		100,  // Slot for light armour (legs)
		100,  // Slot for light armour (torso)
		100,  // Slot for light armour (hands)
		14,   // Slot for light armour (boots)
		15,   // Slot for full armour
		100,  // Slot for emblem
		100,  // Slot for helmet
		100,  // Slot for secondary weapon
		73,   // Slot for primary weapon
		100,  // Slot for shield
		100,  // Unknown slot
		0,    // Gender
		1,    // Character level
		2, 0, // Edged weapons
		1, 0, // Blunted weapons
		1, 0, // Archery
		1, 0, // Polearms
		1, 0, // Wizardry
		0, 0, 0, 0, 0, 0, // Unknown
		117, 115, 101, 114, 0, // User name
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
		0, // Unknown
	}

	// Act
	req := UpdateCharacterStatsRequest(packet[4:])
	data, err := req.Parse()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, packet[4:60], data.CharacterInfo.ToBytes())
	assert.Equal(t, "user", data.User)
	assert.Equal(t, "character", data.Character)
	assert.Equal(t, []byte{0}, data.Unknown)
}
