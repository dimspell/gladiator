package model

import (
	"encoding/binary"
)

type CharacterInfo struct {
	Strength             uint16
	Agility              uint16
	Wisdom               uint16
	Constitution         uint16
	HealthPoints         uint16
	MagicPoints          uint16
	ExperiencePoints     uint32
	Money                uint32
	ScorePoints          uint32
	ClassType            ClassType
	SkinCarnation        SkinCarnation
	HairStyle            HairStyle
	LightArmourLegs      EquipmentSlot
	LightArmourTorso     EquipmentSlot
	LightArmourHands     EquipmentSlot
	LightArmourBoots     EquipmentSlot
	FullArmour           EquipmentSlot
	ArmourEmblem         EquipmentSlot
	Helmet               EquipmentSlot
	SecondaryWeapon      EquipmentSlot // Buggy
	PrimaryWeapon        EquipmentSlot
	Shield               EquipmentSlot
	UnknownEquipmentSlot EquipmentSlot // Unknown
	Gender               Gender
	Level                byte
	EdgedWeapons         uint16
	BluntedWeapons       uint16
	Archery              uint16
	Polearms             uint16
	Wizardry             uint16
	Unknown              []byte // Unknown
}

func NewCharacterInfo(buf []byte) CharacterInfo {
	return CharacterInfo{
		Strength:             binary.LittleEndian.Uint16(buf[0:2]),
		Agility:              binary.LittleEndian.Uint16(buf[2:4]),
		Wisdom:               binary.LittleEndian.Uint16(buf[4:6]),
		Constitution:         binary.LittleEndian.Uint16(buf[6:8]),
		HealthPoints:         binary.LittleEndian.Uint16(buf[8:10]),
		MagicPoints:          binary.LittleEndian.Uint16(buf[10:12]),
		ExperiencePoints:     binary.LittleEndian.Uint32(buf[12:16]),
		Money:                binary.LittleEndian.Uint32(buf[16:20]),
		ScorePoints:          binary.LittleEndian.Uint32(buf[20:24]),
		ClassType:            ClassType(buf[24]),
		SkinCarnation:        SkinCarnation(buf[25]),
		HairStyle:            HairStyle(buf[26]),
		LightArmourLegs:      EquipmentSlot(buf[27]),
		LightArmourTorso:     EquipmentSlot(buf[28]),
		LightArmourHands:     EquipmentSlot(buf[29]),
		LightArmourBoots:     EquipmentSlot(buf[30]),
		FullArmour:           EquipmentSlot(buf[31]),
		ArmourEmblem:         EquipmentSlot(buf[32]),
		Helmet:               EquipmentSlot(buf[33]),
		SecondaryWeapon:      EquipmentSlot(buf[34]),
		PrimaryWeapon:        EquipmentSlot(buf[35]),
		Shield:               EquipmentSlot(buf[36]),
		UnknownEquipmentSlot: EquipmentSlot(buf[37]),
		Gender:               Gender(buf[38]),
		Level:                buf[39],
		EdgedWeapons:         binary.LittleEndian.Uint16(buf[40:42]),
		BluntedWeapons:       binary.LittleEndian.Uint16(buf[42:44]),
		Archery:              binary.LittleEndian.Uint16(buf[44:46]),
		Polearms:             binary.LittleEndian.Uint16(buf[46:48]),
		Wizardry:             binary.LittleEndian.Uint16(buf[48:50]),
		Unknown:              buf[50:56],
	}
}

func (c *CharacterInfo) ToBytes() []byte {
	buf := make([]byte, 56)

	binary.LittleEndian.PutUint16(buf[0:2], c.Strength)
	binary.LittleEndian.PutUint16(buf[2:4], c.Agility)
	binary.LittleEndian.PutUint16(buf[4:6], c.Wisdom)
	binary.LittleEndian.PutUint16(buf[6:8], c.Constitution)
	binary.LittleEndian.PutUint16(buf[8:10], c.HealthPoints)
	binary.LittleEndian.PutUint16(buf[10:12], c.MagicPoints)
	binary.LittleEndian.PutUint32(buf[12:16], c.ExperiencePoints)
	binary.LittleEndian.PutUint32(buf[16:20], c.Money)
	binary.LittleEndian.PutUint32(buf[20:24], c.ScorePoints)

	buf[24] = byte(c.ClassType)
	buf[25] = byte(c.SkinCarnation)
	buf[26] = byte(c.HairStyle)

	buf[27] = byte(c.LightArmourLegs)
	buf[28] = byte(c.LightArmourTorso)
	buf[29] = byte(c.LightArmourHands)
	buf[30] = byte(c.LightArmourBoots)
	buf[31] = byte(c.FullArmour)
	buf[32] = byte(c.ArmourEmblem)
	buf[33] = byte(c.Helmet)
	buf[34] = byte(c.SecondaryWeapon)
	buf[35] = byte(c.PrimaryWeapon)
	buf[36] = byte(c.Shield)
	buf[37] = byte(c.UnknownEquipmentSlot)

	buf[38] = byte(c.Gender)
	buf[39] = c.Level

	binary.LittleEndian.PutUint16(buf[40:42], c.EdgedWeapons)
	binary.LittleEndian.PutUint16(buf[42:44], c.BluntedWeapons)
	binary.LittleEndian.PutUint16(buf[44:46], c.Archery)
	binary.LittleEndian.PutUint16(buf[46:48], c.Polearms)
	binary.LittleEndian.PutUint16(buf[48:50], c.Wizardry)

	copy(buf[50:], c.Unknown)
	return buf
}

type EquipmentSlot byte

func (slot EquipmentSlot) IsEquipped() bool { return slot != 100 }
