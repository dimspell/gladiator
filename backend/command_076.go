package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-76: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	character, err := b.DB.FindCharacter(context.TODO(), database.FindCharacterParams{
		UserID:        session.UserID,
		CharacterName: data.CharacterName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
		}
		return fmt.Errorf("packet-76: no characters found owned by player: %s", err)
	}

	// Provide stats of the selected character
	unknown, _ := base64.StdEncoding.DecodeString(character.Unknown.String)
	info := model.CharacterInfo{
		Strength:             uint16(character.Strength),
		Agility:              uint16(character.Agility),
		Wisdom:               uint16(character.Wisdom),
		Constitution:         uint16(character.Constitution),
		HealthPoints:         uint16(character.HealthPoints),
		MagicPoints:          uint16(character.MagicPoints),
		ExperiencePoints:     uint32(character.ExperiencePoints),
		Money:                uint32(character.Money),
		ScorePoints:          uint32(character.ScorePoints),
		ClassType:            model.ClassType(character.ClassType),
		SkinCarnation:        model.SkinCarnation(character.SkinCarnation),
		HairStyle:            model.HairStyle(character.HairStyle),
		LightArmourLegs:      model.EquipmentSlot(character.LightArmourLegs),
		LightArmourTorso:     model.EquipmentSlot(character.LightArmourTorso),
		LightArmourHands:     model.EquipmentSlot(character.LightArmourHands),
		LightArmourBoots:     model.EquipmentSlot(character.LightArmourBoots),
		FullArmour:           model.EquipmentSlot(character.FullArmour),
		ArmourEmblem:         model.EquipmentSlot(character.ArmourEmblem),
		Helmet:               model.EquipmentSlot(character.Helmet),
		SecondaryWeapon:      model.EquipmentSlot(character.SecondaryWeapon),
		PrimaryWeapon:        model.EquipmentSlot(character.PrimaryWeapon),
		Shield:               model.EquipmentSlot(character.Shield),
		UnknownEquipmentSlot: model.EquipmentSlot(character.UnknownEquipmentSlot),
		Gender:               model.Gender(character.Gender),
		Level:                byte(character.Level),
		EdgedWeapons:         uint16(character.EdgedWeapons),
		BluntedWeapons:       uint16(character.BluntedWeapons),
		Archery:              uint16(character.Archery),
		Polearms:             uint16(character.Polearms),
		Wizardry:             uint16(character.Wizardry),
		Unknown:              unknown,
	}
	response := make([]byte, 60)
	response[0] = 1
	copy(response[4:], info.ToBytes())

	session.CharacterID = character.ID

	return b.Send(session.Conn, SelectCharacter, response)
}

type SelectCharacterRequest []byte

type SelectCharacterRequestData struct {
	Username      string
	CharacterName string
}

func (r SelectCharacterRequest) Parse() (data SelectCharacterRequestData, err error) {
	split := bytes.SplitN(r, []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-76: no enough arguments, malformed request payload: %v", r)
	}
	data.Username = string(split[0])
	data.CharacterName = string(split[1])
	return data, nil
}
