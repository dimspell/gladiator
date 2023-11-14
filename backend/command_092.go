package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-92: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	newCharacter, err := b.DB.CreateCharacter(context.TODO(), database.CreateCharacterParams{
		Strength:             int64(data.CharacterInfo.Strength),
		Agility:              int64(data.CharacterInfo.Agility),
		Wisdom:               int64(data.CharacterInfo.Wisdom),
		Constitution:         int64(data.CharacterInfo.Constitution),
		HealthPoints:         int64(data.CharacterInfo.HealthPoints),
		MagicPoints:          int64(data.CharacterInfo.MagicPoints),
		ExperiencePoints:     int64(data.CharacterInfo.ExperiencePoints),
		Money:                int64(data.CharacterInfo.Money),
		ScorePoints:          int64(data.CharacterInfo.ScorePoints),
		ClassType:            int64(data.CharacterInfo.ClassType),
		SkinCarnation:        int64(data.CharacterInfo.SkinCarnation),
		HairStyle:            int64(data.CharacterInfo.HairStyle),
		LightArmourLegs:      int64(data.CharacterInfo.LightArmourLegs),
		LightArmourTorso:     int64(data.CharacterInfo.LightArmourTorso),
		LightArmourHands:     int64(data.CharacterInfo.LightArmourHands),
		LightArmourBoots:     int64(data.CharacterInfo.LightArmourBoots),
		FullArmour:           int64(data.CharacterInfo.FullArmour),
		ArmourEmblem:         int64(data.CharacterInfo.ArmourEmblem),
		Helmet:               int64(data.CharacterInfo.Helmet),
		SecondaryWeapon:      int64(data.CharacterInfo.SecondaryWeapon),
		PrimaryWeapon:        int64(data.CharacterInfo.PrimaryWeapon),
		Shield:               int64(data.CharacterInfo.Shield),
		UnknownEquipmentSlot: int64(data.CharacterInfo.UnknownEquipmentSlot),
		Gender:               int64(data.CharacterInfo.Gender),
		Level:                int64(data.CharacterInfo.Level),
		EdgedWeapons:         int64(data.CharacterInfo.EdgedWeapons),
		BluntedWeapons:       int64(data.CharacterInfo.BluntedWeapons),
		Archery:              int64(data.CharacterInfo.Archery),
		Polearms:             int64(data.CharacterInfo.Polearms),
		Wizardry:             int64(data.CharacterInfo.Wizardry),
		Unknown: sql.NullString{
			String: base64.StdEncoding.EncodeToString(data.CharacterInfo.Unknown),
			Valid:  true,
		},
		CharacterName: data.CharacterName,
		UserID:        session.UserID,
		SortOrder:     0,
	})
	if err != nil {
		slog.Error("Could not create a character", "err", err)
		return b.Send(session.Conn, CreateCharacter, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-92: new character created", "character", newCharacter.CharacterName, "username", data.Username)

	return b.Send(session.Conn, CreateCharacter, []byte{1, 0, 0, 0})
}

// TODO: check if there is any additional not recognised byte at the end like slot number
type CreateCharacterRequest []byte

type CreateCharacterRequestData struct {
	CharacterInfo model.CharacterInfo
	Username      string
	CharacterName string
}

func (r CreateCharacterRequest) Parse() (data CreateCharacterRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet-92: packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-92: no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.CharacterInfo = model.NewCharacterInfo(r[:56])
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, err
}
