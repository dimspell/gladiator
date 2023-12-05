package memory

// type Memory struct {
// 	channels  []string
// 	gameRooms []model.GameRoom
// 	ranking   model.Ranking
// 	players   []model.User
// }
//
// func (db *Memory) ListChannels() ([]string, error) {
// 	return db.channels, nil
// }
//
// func (db *Memory) GameRooms() []model.GameRoom {
// 	return db.gameRooms
// }
//
// func (db *Memory) Ranking() model.Ranking {
// 	return db.ranking
// }
//
// func (db *Memory) Players() []model.User {
// 	return db.players
// }
//
// func NewMemory() *Memory {
// 	inventory := model.CharacterInventory{
// 		Backpack: [63]model.InventoryItem{
// 			{TypeId: 4, ItemId: 1, Unknown: 17},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 1, ItemId: 15, Unknown: 97},
// 			{TypeId: 1, ItemId: 15, Unknown: 97},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 19},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 33},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 1, ItemId: 8, Unknown: 65},
// 			{TypeId: 11, ItemId: 101, Unknown: 83},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 19},
// 			{TypeId: 11, ItemId: 101, Unknown: 19},
// 			{TypeId: 11, ItemId: 101, Unknown: 51},
// 			{TypeId: 11, ItemId: 101, Unknown: 51},
// 			{TypeId: 11, ItemId: 101, Unknown: 83},
// 			{TypeId: 11, ItemId: 101, Unknown: 83},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 21},
// 			{TypeId: 11, ItemId: 101, Unknown: 21},
// 			{TypeId: 11, ItemId: 101, Unknown: 53},
// 			{TypeId: 11, ItemId: 101, Unknown: 53},
// 			{TypeId: 11, ItemId: 101, Unknown: 85},
// 			{TypeId: 11, ItemId: 101, Unknown: 85},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 21},
// 			{TypeId: 11, ItemId: 101, Unknown: 21},
// 			{TypeId: 11, ItemId: 101, Unknown: 53},
// 			{TypeId: 11, ItemId: 101, Unknown: 53},
// 			{TypeId: 11, ItemId: 101, Unknown: 85},
// 			{TypeId: 11, ItemId: 101, Unknown: 85},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
//
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 			{TypeId: 11, ItemId: 101, Unknown: 121},
// 		},
// 		Belt: [6]model.InventoryItem{
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 			{TypeId: 11, ItemId: 101, Unknown: 97},
// 		},
// 	}
//
// 	player := model.User{
// 		UserName: "user",
// 		Characters: []model.Character{
// 			{
// 				CharacterName: "warrior",
// 				Info: model.CharacterInfo{
// 					Strength:             100,
// 					Agility:              100,
// 					Wisdom:               100,
// 					Constitution:         100,
// 					HealthPoints:         0,
// 					MagicPoints:          0,
// 					ExperiencePoints:     119,
// 					Money:                500000,
// 					ScorePoints:          0,
// 					ClassType:            model.ClassTypeKnight,
// 					SkinCarnation:        model.SkinCarnationMaleBeige,
// 					HairStyle:            model.HairStyleMaleShortBrown,
// 					LightArmourLegs:      2,
// 					LightArmourTorso:     7,
// 					LightArmourHands:     100,
// 					LightArmourBoots:     12,
// 					FullArmour:           100,
// 					ArmourEmblem:         100,
// 					Helmet:               100,
// 					SecondaryWeapon:      100,
// 					PrimaryWeapon:        42,
// 					Shield:               100,
// 					UnknownEquipmentSlot: 100,
// 					Gender:               model.GenderMale,
// 					Level:                1,
// 					EdgedWeapons:         2,
// 					BluntedWeapons:       1,
// 					Archery:              1,
// 					Polearms:             1,
// 					Wizardry:             1,
// 					Unknown:              []byte{0, 0, 0, 0, 0, 0},
// 				},
// 				Inventory: inventory,
// 				Spells:    nil,
// 			},
// 			{
// 				CharacterName: "knight",
// 				Info: model.CharacterInfo{
// 					Strength:             100,
// 					Agility:              100,
// 					Wisdom:               100,
// 					Constitution:         100,
// 					HealthPoints:         0,
// 					MagicPoints:          0,
// 					ExperiencePoints:     119,
// 					Money:                500000,
// 					ScorePoints:          0,
// 					ClassType:            model.ClassTypeKnight,
// 					SkinCarnation:        model.SkinCarnationMaleBeige,
// 					HairStyle:            model.HairStyleMaleShortBrown,
// 					LightArmourLegs:      2,
// 					LightArmourTorso:     7,
// 					LightArmourHands:     100,
// 					LightArmourBoots:     12,
// 					FullArmour:           100,
// 					ArmourEmblem:         100,
// 					Helmet:               100,
// 					SecondaryWeapon:      100,
// 					PrimaryWeapon:        42,
// 					Shield:               100,
// 					UnknownEquipmentSlot: 100,
// 					Gender:               model.GenderMale,
// 					Level:                1,
// 					EdgedWeapons:         2,
// 					BluntedWeapons:       1,
// 					Archery:              1,
// 					Polearms:             1,
// 					Wizardry:             1,
// 					Unknown:              []byte{0, 0, 0, 0, 0, 0},
// 				},
// 				Inventory: inventory,
// 				Spells:    nil,
// 			},
// 			{
// 				CharacterName: "archer",
// 				Info: model.CharacterInfo{
// 					Strength:             100,
// 					Agility:              100,
// 					Wisdom:               100,
// 					Constitution:         100,
// 					HealthPoints:         0,
// 					MagicPoints:          0,
// 					ExperiencePoints:     0,
// 					Money:                500000,
// 					ScorePoints:          0,
// 					ClassType:            model.ClassTypeArcher,
// 					SkinCarnation:        model.SkinCarnationMaleBeige,
// 					HairStyle:            model.HairStyleMaleShortBrown,
// 					LightArmourLegs:      2,
// 					LightArmourTorso:     7,
// 					LightArmourHands:     100,
// 					LightArmourBoots:     12,
// 					FullArmour:           100,
// 					ArmourEmblem:         100,
// 					Helmet:               100,
// 					SecondaryWeapon:      100,
// 					PrimaryWeapon:        66,
// 					Shield:               100,
// 					UnknownEquipmentSlot: 100,
// 					Gender:               model.GenderMale,
// 					Level:                1,
// 					EdgedWeapons:         2,
// 					BluntedWeapons:       1,
// 					Archery:              1,
// 					Polearms:             1,
// 					Wizardry:             1,
// 					Unknown:              []byte{0, 0, 0, 0, 0, 0},
// 				},
// 				Inventory: inventory,
// 				Spells:    nil,
// 			},
// 			{
// 				CharacterName: "mage",
// 				Info: model.CharacterInfo{
// 					Strength:             100,
// 					Agility:              100,
// 					Wisdom:               100,
// 					Constitution:         100,
// 					HealthPoints:         0,
// 					MagicPoints:          0,
// 					ExperiencePoints:     0,
// 					Money:                500000,
// 					ScorePoints:          0,
// 					ClassType:            model.ClassTypeMage,
// 					SkinCarnation:        model.SkinCarnationMaleBeige,
// 					HairStyle:            model.HairStyleMaleShortBrown,
// 					LightArmourLegs:      100,
// 					LightArmourTorso:     100,
// 					LightArmourHands:     100,
// 					LightArmourBoots:     14,
// 					FullArmour:           15,
// 					ArmourEmblem:         100,
// 					Helmet:               100,
// 					SecondaryWeapon:      100,
// 					PrimaryWeapon:        73,
// 					Shield:               100,
// 					UnknownEquipmentSlot: 100,
// 					Gender:               model.GenderMale,
// 					Level:                1,
// 					EdgedWeapons:         2,
// 					BluntedWeapons:       1,
// 					Archery:              1,
// 					Polearms:             1,
// 					Wizardry:             1,
// 					Unknown:              []byte{0, 0, 0, 0, 0, 0},
// 				},
// 				Inventory: inventory,
// 				Spells:    nil,
// 			},
// 		},
// 	}
//
// 	return &Memory{
// 		channels: []string{"channel1", "channel2"},
// 		players:  []model.User{player},
// 		gameRooms: []model.GameRoom{
// 			{
// 				Lobby: model.LobbyRoom{
// 					HostIPAddress: [4]byte{127, 0, 0, 1},
// 					Name:          "Game room",
// 					Password:      "",
// 				},
// 				Players: []model.LobbyPlayer{
// 					{
// 						ClassType: model.ClassTypeMage,
// 						Name:      "Some Mage",
// 						IPAddress: [4]byte{0, 0, 0, 0},
// 					},
// 					{
// 						ClassType: model.ClassTypeArcher,
// 						Name:      "SkeletonArcher",
// 						IPAddress: [4]byte{0, 0, 0, 0},
// 					},
// 				},
// 			},
// 		},
// 		ranking: model.Ranking{
// 			Players: []model.RankingPosition{
// 				{
// 					Rank:          1,
// 					Points:        200,
// 					Username:      "User1",
// 					CharacterName: "Warrior",
// 				},
// 				{
// 					Rank:          2,
// 					Points:        150,
// 					Username:      "Current",
// 					CharacterName: "Mage",
// 				},
// 			},
// 			CurrentPlayer: model.RankingPosition{
// 				Rank:          2,
// 				Points:        150,
// 				Username:      "Current",
// 				CharacterName: "Mage",
// 			},
// 		},
// 	}
// }
