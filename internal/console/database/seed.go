package database

import (
	"context"
	"database/sql"

	"github.com/dimspell/gladiator/internal/console/auth"
	"github.com/dimspell/gladiator/internal/model"
)

func Seed(queries *Queries) error {
	pwd, _ := auth.NewPassword("test")
	user, err := queries.CreateUser(context.Background(), CreateUserParams{
		Username: "archer",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	character, err := queries.CreateCharacter(context.Background(), CreateCharacterParams{
		Strength:         25,
		Agility:          15,
		Wisdom:           11,
		Constitution:     21,
		HealthPoints:     0,
		MagicPoints:      0,
		ExperiencePoints: 0,
		Money:            300,
		ScorePoints:      0,
		ClassType:        int64(model.ClassTypeArcher),
		SkinCarnation:    int64(model.SkinCarnationMaleBeige),
		HairStyle:        int64(model.HairStyleMaleShortWhite),
		LightArmourLegs:  2,
		LightArmourTorso: 7,
		LightArmourHands: 100,
		LightArmourBoots: 12,
		FullArmour:       100,
		ArmourEmblem:     100,
		Helmet:           100,
		SecondaryWeapon:  100,
		PrimaryWeapon:    42,
		Shield:           100,
		ExtraSlot:        100,
		Gender:           int64(model.GenderMale),
		Level:            1,
		EdgedWeapons:     2,
		BluntedWeapons:   1,
		Archery:          1,
		Polearms:         1,
		Wizardry:         1,
		BonusPoints:      100,
		CharacterName:    "archer",
		UserID:           user.ID,
	})
	if err != nil {
		return err
	}

	// Synthetic spell placeholder (43 uniform bytes, base64). Length matches
	// the wire-compatible record size; content is not copied game data.
	_ = queries.UpdateCharacterSpells(context.Background(), UpdateCharacterSpellsParams{
		CharacterName: character.CharacterName,
		Spells: sql.NullString{
			String: "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQ==",
			Valid:  true,
		},
		UserID: user.ID,
	})

	user2, err := queries.CreateUser(context.Background(), CreateUserParams{
		Username: "mage",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	character2, err := queries.CreateCharacter(context.Background(), CreateCharacterParams{
		Strength:         15,
		Agility:          10,
		Wisdom:           30,
		Constitution:     15,
		HealthPoints:     0,
		MagicPoints:      0,
		ExperiencePoints: 0,
		Money:            300,
		ScorePoints:      0,
		ClassType:        int64(model.ClassTypeMage),
		SkinCarnation:    int64(model.SkinCarnationFemaleLightBrown),
		HairStyle:        int64(model.HairStyleFemaleLongBrown), // 129
		LightArmourLegs:  100,
		LightArmourTorso: 100,
		LightArmourHands: 100,
		LightArmourBoots: 14,
		FullArmour:       15,
		ArmourEmblem:     100,
		Helmet:           100,
		SecondaryWeapon:  100,
		PrimaryWeapon:    73,
		Shield:           100,
		ExtraSlot:        100,
		Gender:           int64(model.GenderFemale),
		Level:            1,
		EdgedWeapons:     1,
		BluntedWeapons:   1,
		Archery:          1,
		Polearms:         1,
		Wizardry:         2,
		BonusPoints:      10,
		CharacterName:    "mage",
		UserID:           user2.ID,
	})
	if err != nil {
		return err
	}

	// Synthetic spell placeholder (43 uniform bytes, base64).
	_ = queries.UpdateCharacterSpells(context.Background(), UpdateCharacterSpellsParams{
		CharacterName: character2.CharacterName,
		Spells: sql.NullString{
			String: "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAg==",
			Valid:  true,
		},
		UserID: user2.ID,
	})

	// Additional test users for multi-player scenarios
	user3, err := queries.CreateUser(context.Background(), CreateUserParams{
		Username: "warrior",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	_, err = queries.CreateCharacter(context.Background(), CreateCharacterParams{
		Strength:         30,
		Agility:          12,
		Wisdom:           8,
		Constitution:     25,
		HealthPoints:     0,
		MagicPoints:      0,
		ExperiencePoints: 0,
		Money:            300,
		ScorePoints:      0,
		ClassType:        int64(model.ClassTypeWarrior),
		SkinCarnation:    int64(model.SkinCarnationMaleBeige),
		HairStyle:        int64(model.HairStyleMaleShortBlack),
		LightArmourLegs:  100,
		LightArmourTorso: 100,
		LightArmourHands: 100,
		LightArmourBoots: 100,
		FullArmour:       10,
		ArmourEmblem:     100,
		Helmet:           100,
		SecondaryWeapon:  100,
		PrimaryWeapon:        20,
		Shield:               5,
		UnknownEquipmentSlot: 100,
		Gender:               int64(model.GenderMale),
		Level:                1,
		EdgedWeapons:         2,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:         1,
		BonusPoints:      50,
		CharacterName:    "warrior",
		UserID:           user3.ID,
	})
	if err != nil {
		return err
	}

	user4, err := queries.CreateUser(context.Background(), CreateUserParams{
		Username: "necro",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	_, err = queries.CreateCharacter(context.Background(), CreateCharacterParams{
		Strength:         12,
		Agility:          15,
		Wisdom:           28,
		Constitution:     18,
		HealthPoints:     0,
		MagicPoints:      0,
		ExperiencePoints: 0,
		Money:            300,
		ScorePoints:      0,
		ClassType:        int64(model.ClassTypeMage),
		SkinCarnation:    int64(model.SkinCarnationFemaleLightBrown),
		HairStyle:        int64(model.HairStyleFemaleLongBlack),
		LightArmourLegs:  100,
		LightArmourTorso: 100,
		LightArmourHands: 100,
		LightArmourBoots: 100,
		FullArmour:       100,
		ArmourEmblem:     100,
		Helmet:           100,
		SecondaryWeapon:  100,
		PrimaryWeapon:        35,
		Shield:               100,
		UnknownEquipmentSlot: 100,
		Gender:               int64(model.GenderFemale),
		Level:                1,
		EdgedWeapons:     1,
		BluntedWeapons:   1,
		Archery:          1,
		Polearms:         1,
		Wizardry:         2,
		BonusPoints:      30,
		CharacterName:    "necro",
		UserID:           user4.ID,
	})
	if err != nil {
		return err
	}

	return nil
}
