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
		return fmt.Errorf("packet-108: could not parse request: %w", err)
	}

	if err := b.DB.UpdateCharacterStats(context.TODO(), database.UpdateCharacterStatsParams{
		Strength:             int64(data.ParsedInfo.Strength),
		Agility:              int64(data.ParsedInfo.Agility),
		Wisdom:               int64(data.ParsedInfo.Wisdom),
		Constitution:         int64(data.ParsedInfo.Constitution),
		HealthPoints:         int64(data.ParsedInfo.HealthPoints),
		MagicPoints:          int64(data.ParsedInfo.MagicPoints),
		ExperiencePoints:     int64(data.ParsedInfo.ExperiencePoints),
		Money:                int64(data.ParsedInfo.Money),
		ScorePoints:          int64(data.ParsedInfo.ScorePoints),
		ClassType:            int64(data.ParsedInfo.ClassType),
		SkinCarnation:        int64(data.ParsedInfo.SkinCarnation),
		HairStyle:            int64(data.ParsedInfo.HairStyle),
		LightArmourLegs:      int64(data.ParsedInfo.LightArmourLegs),
		LightArmourTorso:     int64(data.ParsedInfo.LightArmourTorso),
		LightArmourHands:     int64(data.ParsedInfo.LightArmourHands),
		LightArmourBoots:     int64(data.ParsedInfo.LightArmourBoots),
		FullArmour:           int64(data.ParsedInfo.FullArmour),
		ArmourEmblem:         int64(data.ParsedInfo.ArmourEmblem),
		Helmet:               int64(data.ParsedInfo.Helmet),
		SecondaryWeapon:      int64(data.ParsedInfo.SecondaryWeapon),
		PrimaryWeapon:        int64(data.ParsedInfo.PrimaryWeapon),
		Shield:               int64(data.ParsedInfo.Shield),
		UnknownEquipmentSlot: int64(data.ParsedInfo.UnknownEquipmentSlot),
		Gender:               int64(data.ParsedInfo.Gender),
		Level:                int64(data.ParsedInfo.Level),
		EdgedWeapons:         int64(data.ParsedInfo.EdgedWeapons),
		BluntedWeapons:       int64(data.ParsedInfo.BluntedWeapons),
		Archery:              int64(data.ParsedInfo.Archery),
		Polearms:             int64(data.ParsedInfo.Polearms),
		Wizardry:             int64(data.ParsedInfo.Wizardry),
		Unknown:              sql.NullString{Valid: true, String: base64.StdEncoding.EncodeToString(data.ParsedInfo.Unknown)},
		CharacterName:        data.Character,
		UserID:               session.UserID,
	}); err != nil {
		return err
	}

	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte

type UpdateCharacterStatsRequestData struct {
	Info       []byte
	ParsedInfo model.CharacterInfo
	User       string
	Character  string
	Unknown    []byte
}

func (r UpdateCharacterStatsRequest) Parse() (data UpdateCharacterStatsRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.Info = r[:56]
	data.ParsedInfo = model.ParseCharacterInfo(r[:56])
	data.User = string(split[0])
	data.Character = string(split[1])
	data.Unknown = split[2]

	return data, err
}
