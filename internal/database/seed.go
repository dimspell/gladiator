package database

import (
	"context"
	"database/sql"

	"github.com/dispel-re/dispel-multi/model"
	"golang.org/x/crypto/bcrypt"
)

func Seed(queries *Queries) error {
	pwd, _ := bcrypt.GenerateFromPassword([]byte("test"), 14)
	user, err := queries.CreateUser(context.TODO(), CreateUserParams{
		Username: "test",
		Password: string(pwd),
	})
	if err != nil {
		return err
	}

	_, err = queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             100,
		Agility:              100,
		Wisdom:               100,
		Constitution:         100,
		HealthPoints:         100,
		MagicPoints:          100,
		ExperiencePoints:     9,
		Money:                300,
		ScorePoints:          0,
		ClassType:            int64(model.ClassTypeArcher),
		SkinCarnation:        int64(model.SkinCarnationMaleBrown),
		HairStyle:            int64(model.HairStyleMaleLongGray),
		LightArmourLegs:      100,
		LightArmourTorso:     100,
		LightArmourHands:     100,
		LightArmourBoots:     100,
		FullArmour:           100,
		ArmourEmblem:         100,
		Helmet:               100,
		SecondaryWeapon:      100,
		PrimaryWeapon:        100,
		Shield:               100,
		UnknownEquipmentSlot: 100,
		Gender:               int64(model.GenderMale),
		Level:                1,
		EdgedWeapons:         1,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:             1,
		Unknown:              sql.NullString{Valid: true, String: "\x00\x00\x00\x00\x00\x00"},
		CharacterName:        "tester",
		UserID:               user.ID,
		SortOrder:            0,
	})
	if err != nil {
		return err
	}

	user2, err := queries.CreateUser(context.TODO(), CreateUserParams{
		Username: "tester",
		Password: string(pwd),
	})
	if err != nil {
		return err
	}

	_, err = queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             100,
		Agility:              100,
		Wisdom:               100,
		Constitution:         100,
		HealthPoints:         100,
		MagicPoints:          100,
		ExperiencePoints:     9,
		Money:                300,
		ScorePoints:          0,
		ClassType:            int64(model.ClassTypeArcher),
		SkinCarnation:        int64(model.SkinCarnationMaleBrown),
		HairStyle:            int64(model.HairStyleMaleLongGray),
		LightArmourLegs:      100,
		LightArmourTorso:     100,
		LightArmourHands:     100,
		LightArmourBoots:     100,
		FullArmour:           100,
		ArmourEmblem:         100,
		Helmet:               100,
		SecondaryWeapon:      100,
		PrimaryWeapon:        100,
		Shield:               100,
		UnknownEquipmentSlot: 100,
		Gender:               int64(model.GenderMale),
		Level:                1,
		EdgedWeapons:         1,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:             1,
		Unknown:              sql.NullString{Valid: true, String: "\x00\x00\x00\x00\x00\x00"},
		CharacterName:        "character",
		UserID:               user2.ID,
		SortOrder:            0,
	})
	if err != nil {
		return err
	}

	return nil
}
