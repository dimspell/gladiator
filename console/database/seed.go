package database

import (
	"context"
	"database/sql"
	"encoding/base64"

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

	spells := make([]byte, 43)
	for i := 0; i < 41; i++ {
		spells[i] = 2
	}

	inventory := model.CharacterInventory{
		Backpack: [63]model.InventoryItem{
			{TypeId: 4, ItemId: 1, Unknown: 17}, // Money
			{TypeId: 2, ItemId: 2, Unknown: 33}, // HP Potion
			{TypeId: 2, ItemId: 2, Unknown: 49},
			{TypeId: 2, ItemId: 2, Unknown: 65},
			{TypeId: 2, ItemId: 2, Unknown: 81},
			{TypeId: 2, ItemId: 2, Unknown: 97},
			{TypeId: 2, ItemId: 2, Unknown: 113},

			{TypeId: 2, ItemId: 4, Unknown: 18}, // MP Potion
			{TypeId: 2, ItemId: 4, Unknown: 34},
			{TypeId: 2, ItemId: 4, Unknown: 50},
			{TypeId: 2, ItemId: 4, Unknown: 66},
			{TypeId: 2, ItemId: 4, Unknown: 82},
			{TypeId: 2, ItemId: 4, Unknown: 98},
			{TypeId: 2, ItemId: 4, Unknown: 114},

			{TypeId: 2, ItemId: 5, Unknown: 19}, // Antidote
			{TypeId: 2, ItemId: 5, Unknown: 35},
			{TypeId: 2, ItemId: 5, Unknown: 51},
			{TypeId: 2, ItemId: 5, Unknown: 67},
			{TypeId: 2, ItemId: 5, Unknown: 83},
			{TypeId: 2, ItemId: 5, Unknown: 99},
			{TypeId: 2, ItemId: 5, Unknown: 115},

			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},

			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},

			{TypeId: 11, ItemId: 101, Unknown: 21},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},

			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},

			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},

			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
			{TypeId: 11, ItemId: 101, Unknown: 121},
		},
		Belt: [6]model.InventoryItem{
			{TypeId: 2, ItemId: 2, Unknown: 17},
			{TypeId: 2, ItemId: 2, Unknown: 33},
			{TypeId: 2, ItemId: 2, Unknown: 49},
			{TypeId: 2, ItemId: 4, Unknown: 65},
			{TypeId: 2, ItemId: 4, Unknown: 81},
			{TypeId: 11, ItemId: 101, Unknown: 97},
		},
	}

	character, err := queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             100,
		Agility:              100,
		Wisdom:               100,
		Constitution:         100,
		HealthPoints:         100,
		MagicPoints:          100,
		ExperiencePoints:     9,
		Money:                300000,
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
		Wizardry:             100,
		BonusPoints:          10,
		CharacterName:        "tester",
		UserID:               user.ID,
	})
	if err != nil {
		return err
	}

	queries.UpdateCharacterSpells(context.TODO(), UpdateCharacterSpellsParams{
		CharacterName: character.CharacterName,
		Spells: sql.NullString{
			String: base64.StdEncoding.EncodeToString(spells),
			Valid:  true,
		},
		UserID: user.ID,
	})

	queries.UpdateCharacterInventory(context.TODO(), UpdateCharacterInventoryParams{
		CharacterName: character.CharacterName,
		Inventory: sql.NullString{
			String: base64.StdEncoding.EncodeToString(inventory.ToBytes()),
			Valid:  true,
		},
		UserID: user.ID,
	})

	user2, err := queries.CreateUser(context.TODO(), CreateUserParams{
		Username: "tester",
		Password: string(pwd),
	})
	if err != nil {
		return err
	}

	character2, err := queries.CreateCharacter(context.TODO(), CreateCharacterParams{
		Strength:             100,
		Agility:              100,
		Wisdom:               100,
		Constitution:         100,
		HealthPoints:         100,
		MagicPoints:          100,
		ExperiencePoints:     9,
		Money:                300000,
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
		Wizardry:             100,
		BonusPoints:          10,
		CharacterName:        "character2",
		UserID:               user2.ID,
	})
	if err != nil {
		return err
	}

	queries.UpdateCharacterSpells(context.TODO(), UpdateCharacterSpellsParams{
		CharacterName: character2.CharacterName,
		Spells: sql.NullString{
			String: base64.StdEncoding.EncodeToString(spells),
			Valid:  true,
		},
		UserID: user2.ID,
	})

	queries.UpdateCharacterInventory(context.TODO(), UpdateCharacterInventoryParams{
		CharacterName: character2.CharacterName,
		Inventory: sql.NullString{
			String: base64.StdEncoding.EncodeToString(inventory.ToBytes()),
			Valid:  true,
		},
		UserID: user2.ID,
	})

	gameRoom, _ := queries.CreateGameRoom(context.TODO(), CreateGameRoomParams{
		Name:          "test",
		Password:      sql.NullString{Valid: false},
		MapID:         1,
		HostIpAddress: "127.0.0.28",
		// UserID:   user2.ID,
	})

	queries.AddPlayerToRoom(context.TODO(), AddPlayerToRoomParams{
		GameRoomID:  gameRoom.ID,
		CharacterID: character.ID,
		IpAddress:   "127.0.0.28",
	})

	// respGame, err := b.GameClient.CreateGame(context.TODO(), connect.NewRequest(&multiv1.CreateGameRequest{
	// 	UserId:        session.UserID,
	// 	GameName:      data.RoomName,
	// 	Password:      data.Password,
	// 	HostIpAddress: hostIPAddress,
	// 	MapId:         int64(data.MapID),
	// }))
	// if err != nil {
	// 	return err
	// }
	// slog.Info("packet-28: created game room",
	// 	"id", respGame.Msg.Game.GameId,
	// 	"name", respGame.Msg.Game.Name)

	// _, err = b.GameClient.JoinGame(context.TODO(), connect.NewRequest(&multiv1.JoinGameRequest{
	// 	UserId:      session.UserID,
	// 	CharacterId: session.CharacterID,
	// 	GameRoomId:  respGame.Msg.Game.GetGameId(),
	// 	IpAddress:   hostIPAddress,
	// }))
	// if err != nil {
	// 	return err
	// }

	return nil
}
