package database

import (
	"context"
	"database/sql"

	"github.com/dimspell/gladiator/internal/console/auth"
	"github.com/dimspell/gladiator/internal/model"
)

func Seed(queries *Queries) error {
	pwd, _ := auth.NewPassword("test")
	user, err := queries.CreateUser(context.TODO(), CreateUserParams{
		Username: "archer",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	// spells := make([]byte, 43)
	// for i := 0; i < 41; i++ {
	// 	spells[i] = 2
	// }

	// inventory := model.CharacterInventory{
	// 	Backpack: [63]model.InventoryItem{
	// 		{TypeId: 4, ItemId: 1, Unknown: 17}, // Money
	// 		{TypeId: 2, ItemId: 2, Unknown: 33}, // HP Potion
	// 		{TypeId: 2, ItemId: 2, Unknown: 49},
	// 		{TypeId: 2, ItemId: 2, Unknown: 65},
	// 		{TypeId: 2, ItemId: 2, Unknown: 81},
	// 		{TypeId: 2, ItemId: 2, Unknown: 97},
	// 		{TypeId: 2, ItemId: 2, Unknown: 113},

	// 		{TypeId: 2, ItemId: 4, Unknown: 18}, // MP Potion
	// 		{TypeId: 2, ItemId: 4, Unknown: 34},
	// 		{TypeId: 2, ItemId: 4, Unknown: 50},
	// 		{TypeId: 2, ItemId: 4, Unknown: 66},
	// 		{TypeId: 2, ItemId: 4, Unknown: 82},
	// 		{TypeId: 2, ItemId: 4, Unknown: 98},
	// 		{TypeId: 2, ItemId: 4, Unknown: 114},

	// 		{TypeId: 2, ItemId: 5, Unknown: 19}, // Antidote
	// 		{TypeId: 2, ItemId: 5, Unknown: 35},
	// 		{TypeId: 2, ItemId: 5, Unknown: 51},
	// 		{TypeId: 2, ItemId: 5, Unknown: 67},
	// 		{TypeId: 2, ItemId: 5, Unknown: 83},
	// 		{TypeId: 2, ItemId: 5, Unknown: 99},
	// 		{TypeId: 2, ItemId: 5, Unknown: 115},

	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},

	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},

	// 		{TypeId: 11, ItemId: 101, Unknown: 21},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},

	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},

	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},

	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 	},
	// 	Belt: [6]model.InventoryItem{
	// 		{TypeId: 2, ItemId: 2, Unknown: 17},
	// 		{TypeId: 2, ItemId: 2, Unknown: 33},
	// 		{TypeId: 2, ItemId: 2, Unknown: 49},
	// 		{TypeId: 2, ItemId: 4, Unknown: 65},
	// 		{TypeId: 2, ItemId: 4, Unknown: 81},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 	},
	// }

	character, err := queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             25,
		Agility:              15,
		Wisdom:               11,
		Constitution:         21,
		HealthPoints:         0,
		MagicPoints:          0,
		ExperiencePoints:     0,
		Money:                300,
		ScorePoints:          0,
		ClassType:            int64(model.ClassTypeArcher),
		SkinCarnation:        int64(model.SkinCarnationMaleBeige),
		HairStyle:            int64(model.HairStyleMaleShortWhite),
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
		Gender:               int64(model.GenderMale),
		Level:                1,
		EdgedWeapons:         2,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:             1,
		BonusPoints:          100,
		CharacterName:        "archer",
		UserID:               user.ID,
	})
	if err != nil {
		return err
	}

	queries.UpdateCharacterSpells(context.TODO(), UpdateCharacterSpellsParams{
		CharacterName: character.CharacterName,
		Spells: sql.NullString{
			String: "AQEBAQEBAQEBAQEBAQEBAgEBAQIBAQEBAQEBAQEBAQEBAQEBAQEBAQEAAA==",
			Valid:  true,
		},
		UserID: user.ID,
	})

	user2, err := queries.CreateUser(context.TODO(), CreateUserParams{
		Username: "mage",
		Password: pwd.String(),
	})
	if err != nil {
		return err
	}

	character2, err := queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             15,
		Agility:              10,
		Wisdom:               30,
		Constitution:         15,
		HealthPoints:         0,
		MagicPoints:          0,
		ExperiencePoints:     0,
		Money:                300,
		ScorePoints:          0,
		ClassType:            int64(model.ClassTypeMage),
		SkinCarnation:        int64(model.SkinCarnationFemaleLightBrown),
		HairStyle:            int64(model.HairStyleFemaleLongBrown), // 129
		LightArmourLegs:      100,
		LightArmourTorso:     100,
		LightArmourHands:     100,
		LightArmourBoots:     14,
		FullArmour:           15,
		ArmourEmblem:         100,
		Helmet:               100,
		SecondaryWeapon:      100,
		PrimaryWeapon:        73,
		Shield:               100,
		UnknownEquipmentSlot: 100,
		Gender:               int64(model.GenderFemale),
		Level:                1,
		EdgedWeapons:         1,
		BluntedWeapons:       1,
		Archery:              1,
		Polearms:             1,
		Wizardry:             2,
		BonusPoints:          10,
		CharacterName:        "mage",
		UserID:               user2.ID,
	})
	if err != nil {
		return err
	}

	queries.UpdateCharacterSpells(context.TODO(), UpdateCharacterSpellsParams{
		CharacterName: character2.CharacterName,
		Spells: sql.NullString{
			String: "AgICAgECAgEBAQIBAQIBAQIBAQEBAQECAQIBAQEBAQEBAQEBAQEBAQEAAA==",
			Valid:  true,
		},
		UserID: user2.ID,
	})

	// queries.UpdateCharacterInventory(context.TODO(), UpdateCharacterInventoryParams{
	// 	CharacterName: character2.CharacterName,
	// 	Inventory: sql.NullString{
	// 		String: base64.StdEncoding.EncodeToString(inventory.ToBytes()),
	// 		Valid:  true,
	// 	},
	// 	CreatorID: user2.ID,
	// })
	//
	// gameRoom, err := queries.CreateGameRoom(context.TODO(), CreateGameRoomParams{
	// 	Name:          "test",
	// 	Password:      sql.NullString{Valid: false},
	// 	MapID:         2,
	// 	HostIpAddress: "192.168.121.169",
	// 	// CreatorID:   user2.ID,
	// })
	// if err != nil {
	// 	return err
	// }
	// queries.AddPlayerToRoom(context.TODO(), AddPlayerToRoomParams{
	// 	GameRoomID:  gameRoom.ID,
	// 	CharacterID: character2.ID,
	// 	IpAddress:   "192.168.121.169",
	// })
	// queries.AddPlayerToRoom(context.TODO(), AddPlayerToRoomParams{
	// 	GameRoomID:  gameRoom.ID,
	// 	CharacterID: character.ID,
	// 	IpAddress:   "192.168.121.212",
	// })

	return nil
}
