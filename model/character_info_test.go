package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCharacterInfo(t *testing.T) {
	// Arrange
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
		117, 115, 101, 114, 0, // Username
		99, 104, 97, 114, 97, 99, 116, 101, 114, 0, // Character name
	}

	// Act
	info := NewCharacterInfo(packet[4:])

	// Assert
	assert.Equal(t, uint16(20), info.Strength)
	assert.Equal(t, uint16(15), info.Agility)
	assert.Equal(t, uint16(11), info.Wisdom)
	assert.Equal(t, uint16(21), info.Constitution)
	assert.Equal(t, uint16(0), info.HealthPoints)
	assert.Equal(t, uint16(0), info.MagicPoints)
	assert.Equal(t, uint32(0), info.ExperiencePoints)
	assert.Equal(t, uint32(300), info.Money)
	assert.Equal(t, uint32(0), info.ScorePoints)
	assert.Equal(t, ClassTypeKnight, info.ClassType)
	assert.Equal(t, SkinCarnationMaleBeige, info.SkinCarnation)
	assert.Equal(t, HairStyleMaleShortWhite, info.HairStyle)
	assert.Equal(t, EquipmentSlot(2), info.LightArmourLegs)
	assert.Equal(t, EquipmentSlot(7), info.LightArmourTorso)
	assert.Equal(t, EquipmentSlot(100), info.LightArmourHands)
	assert.Equal(t, EquipmentSlot(12), info.LightArmourBoots)
	assert.Equal(t, EquipmentSlot(100), info.FullArmour)
	assert.Equal(t, EquipmentSlot(100), info.ArmourEmblem)
	assert.Equal(t, EquipmentSlot(100), info.Helmet)
	assert.Equal(t, EquipmentSlot(100), info.SecondaryWeapon)
	assert.Equal(t, EquipmentSlot(42), info.PrimaryWeapon)
	assert.Equal(t, EquipmentSlot(100), info.Shield)
	assert.Equal(t, EquipmentSlot(100), info.UnknownEquipmentSlot)
	assert.Equal(t, GenderMale, info.Gender)
	assert.Equal(t, byte(1), info.Level)
	assert.Equal(t, uint16(1), info.EdgedWeapons)
	assert.Equal(t, uint16(2), info.BluntedWeapons)
	assert.Equal(t, uint16(1), info.Archery)
	assert.Equal(t, uint16(1), info.Polearms)
	assert.Equal(t, uint16(1), info.Wizardry)
	assert.Equal(t, []byte{0, 0, 0, 0, 0, 0}, info.Unknown)
	assert.True(t, bytes.Equal(packet[4:60], info.ToBytes()))
}

func BenchmarkCharacterInfo_ToBytes(b *testing.B) {
	b.StopTimer()
	into := CharacterInfo{
		Strength:             20,
		Agility:              15,
		Wisdom:               11,
		Constitution:         21,
		HealthPoints:         0,
		MagicPoints:          0,
		ExperiencePoints:     119,
		Money:                300,
		ScorePoints:          0,
		ClassType:            ClassTypeKnight,
		SkinCarnation:        SkinCarnationMaleBeige,
		HairStyle:            HairStyleMaleShortBrown,
		LightArmourLegs:      2,
		LightArmourTorso:     7,
		LightArmourHands:     100,
		LightArmourBoots:     12,
		FullArmour:           100,
		ArmourEmblem:         100,
		Helmet:               100,
		SecondaryWeapon:      100,
		PrimaryWeapon:        42,
		Shield:               100,
		UnknownEquipmentSlot: 100,
		Gender:               GenderMale,
		Level:                1,
		EdgedWeapons:         2,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:             1,
		Unknown:              []byte{0, 0, 0, 0, 0, 0},
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		into.ToBytes()
	}
}
