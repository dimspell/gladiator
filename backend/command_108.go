package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleUpdateCharacterStats handles 0x6cff (255-108) command.
//
// It can be received by the game server in multiple scenarios:
//   - .
func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-108: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	if err := b.DB.UpdateCharacterStats(context.TODO(), database.UpdateCharacterStatsParams{
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
		Unknown:              sql.NullString{Valid: true, String: base64.StdEncoding.EncodeToString(data.CharacterInfo.Unknown)},
		CharacterName:        data.Character,
		UserID:               session.UserID,
	}); err != nil {
		return err
	}

	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte

type UpdateCharacterStatsRequestData struct {
	CharacterInfo model.CharacterInfo
	User          string
	Character     string
	Unknown       []byte
}

func (r UpdateCharacterStatsRequest) Parse() (data UpdateCharacterStatsRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet-108: packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-108: no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.CharacterInfo = model.NewCharacterInfo(r[:56])
	data.User = string(split[0])
	data.Character = string(split[1])
	data.Unknown = split[2]

	return data, err
}
