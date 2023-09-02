package backend

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
