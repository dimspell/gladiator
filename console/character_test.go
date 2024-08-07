package console

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/console/database"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func setupDatabase(t testing.TB) *database.SQLite {
	t.Helper()

	db, err := database.NewMemory()
	if err != nil {
		panic(err)
	}
	if err := db.Ping(); err != nil {
		panic(err)
	}

	t.Cleanup(func() {
		if db != nil {
			_ = db.Close()
		}
	})

	return db
}

func TestCharacterServiceServer_ListCharacters(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("No user", func(t *testing.T) {
		// Arrange
		db := setupDatabase(t)
		defer db.Close()

		// Act
		resp, err := (&characterServiceServer{
			DB: db,
		}).ListCharacters(context.Background(), &connect.Request[multiv1.ListCharactersRequest]{
			Msg: &multiv1.ListCharactersRequest{
				UserId: 404,
			},
		})

		var connectError *connect.Error
		errors.As(err, &connectError)
		assert.Equal(t, connect.CodeNotFound, connectError.Code())
		assert.ErrorContains(t, err, "user not found")
		assert.Nil(t, resp)
	})

	t.Run("No characters", func(t *testing.T) {
		// Arrange
		db := setupDatabase(t)
		defer db.Close()

		if _, err := db.Writer.Exec("INSERT INTO users (id, username, password) VALUES (15, 'test', '<PASSWORD>')"); err != nil {
			t.Fatalf("could not insert user: %v", err)
		}

		// Act
		resp, err := (&characterServiceServer{
			DB: db,
		}).ListCharacters(context.Background(), &connect.Request[multiv1.ListCharactersRequest]{
			Msg: &multiv1.ListCharactersRequest{
				UserId: 15,
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp.Msg)
		assert.Empty(t, resp.Msg.GetCharacters())
	})

	t.Run("Two characters", func(t *testing.T) {
		// Arrange
		db := setupDatabase(t)
		defer db.Close()

		if _, err := db.Writer.Exec("INSERT INTO users (id, username, password) VALUES (10, 'test', '<PASSWORD>')"); err != nil {
			t.Fatalf("could not insert user: %v", err)
		}
		if _, err := db.Writer.Exec(`INSERT INTO characters (id, user_id, character_name, strength, agility, wisdom, constitution, health_points,
                        magic_points, experience_points, money, score_points, class_type, skin_carnation, hair_style,
                        light_armour_legs, light_armour_torso, light_armour_hands, light_armour_boots, full_armour,
                        armour_emblem, helmet, secondary_weapon, primary_weapon, shield, unknown_equipment_slot, gender,
                        level, edged_weapons, blunted_weapons, archery, polearms, wizardry, holy_magic, dark_magic,
                        bonus_points, inventory, spells)
						VALUES (100, 10, 'archer', 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, NULL, NULL);`); err != nil {
			t.Fatalf("could not insert character: %v", err)
		}
		if _, err := db.Writer.Exec(`INSERT INTO characters (id, user_id, character_name, strength, agility, wisdom, constitution, health_points,
                        magic_points, experience_points, money, score_points, class_type, skin_carnation, hair_style,
                        light_armour_legs, light_armour_torso, light_armour_hands, light_armour_boots, full_armour,
                        armour_emblem, helmet, secondary_weapon, primary_weapon, shield, unknown_equipment_slot, gender,
                        level, edged_weapons, blunted_weapons, archery, polearms, wizardry, holy_magic, dark_magic,
                        bonus_points, inventory, spells)
						VALUES (101, 10, 'mage', 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, NULL, NULL)`); err != nil {
			t.Fatalf("could not insert character: %v", err)
		}

		// Act
		resp, err := (&characterServiceServer{
			DB: db,
		}).ListCharacters(context.Background(), &connect.Request[multiv1.ListCharactersRequest]{
			Msg: &multiv1.ListCharactersRequest{
				UserId: 10,
			},
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp.Msg)
		assert.Len(t, resp.Msg.Characters, 2)
		assert.Equal(t, "archer", resp.Msg.Characters[0].CharacterName)
		assert.Equal(t, "mage", resp.Msg.Characters[1].CharacterName)

		stats := model.ParseCharacterInfo(resp.Msg.Characters[0].Stats)
		assert.Equalf(t, uint16(1), stats.Strength, "stats.Strength")
		assert.Equalf(t, uint16(2), stats.Agility, "stats.Agility")
		assert.Equalf(t, uint16(3), stats.Wisdom, "stats.Wisdom")
		assert.Equalf(t, uint16(4), stats.Constitution, "stats.Constitution")
		assert.Equalf(t, uint16(5), stats.HealthPoints, "stats.HealthPoints")
		assert.Equalf(t, uint16(6), stats.MagicPoints, "stats.MagicPoints")
		assert.Equalf(t, uint32(7), stats.ExperiencePoints, "stats.ExperiencePoints")
		assert.Equalf(t, uint32(8), stats.Money, "stats.Money")
		assert.Equalf(t, uint32(9), stats.ScorePoints, "stats.ScorePoints")
		assert.Equalf(t, uint16(33), stats.BonusPoints, "stats.BonusPoints")
		assert.Equalf(t, model.ClassType(10), stats.ClassType, "stats.ClassType")
		assert.Equalf(t, model.SkinCarnation(11), stats.SkinCarnation, "stats.SkinCarnation")
		assert.Equalf(t, model.HairStyle(12), stats.HairStyle, "stats.HairStyle")
		assert.Equalf(t, model.EquipmentSlot(13), stats.LightArmourLegs, "stats.LightArmourLegs")
		assert.Equalf(t, model.EquipmentSlot(14), stats.LightArmourTorso, "stats.LightArmourTorso")
		assert.Equalf(t, model.EquipmentSlot(15), stats.LightArmourHands, "stats.LightArmourHands")
		assert.Equalf(t, model.EquipmentSlot(16), stats.LightArmourBoots, "stats.LightArmourBoots")
		assert.Equalf(t, model.EquipmentSlot(17), stats.FullArmour, "stats.FullArmour")
		assert.Equalf(t, model.EquipmentSlot(18), stats.ArmourEmblem, "stats.ArmourEmblem")
		assert.Equalf(t, model.EquipmentSlot(19), stats.Helmet, "stats.Helmet")
		assert.Equalf(t, model.EquipmentSlot(20), stats.SecondaryWeapon, "stats.SecondaryWeapon")
		assert.Equalf(t, model.EquipmentSlot(21), stats.PrimaryWeapon, "stats.PrimaryWeapon")
		assert.Equalf(t, model.EquipmentSlot(22), stats.Shield, "stats.Shield")
		assert.Equalf(t, model.EquipmentSlot(23), stats.UnknownEquipmentSlot, "stats.UnknownEquipmentSlot")
		assert.Equalf(t, model.Gender(24), stats.Gender, "stats.Gender")
		assert.Equalf(t, uint8(25), stats.Level, "stats.Level")
		assert.Equalf(t, uint16(26), stats.EdgedWeapons, "stats.EdgedWeapons")
		assert.Equalf(t, uint16(27), stats.BluntedWeapons, "stats.BluntedWeapons")
		assert.Equalf(t, uint16(28), stats.Archery, "stats.Archery")
		assert.Equalf(t, uint16(29), stats.Polearms, "stats.Polearms")
		assert.Equalf(t, uint16(30), stats.Wizardry, "stats.Wizardry")
		assert.Equalf(t, uint16(31), stats.HolyMagic, "stats.HolyMagic")
		assert.Equalf(t, uint16(32), stats.DarkMagic, "stats.DarkMagic")
	})
}
