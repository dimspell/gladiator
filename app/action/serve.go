package action

import (
	"context"
	"database/sql"

	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/urfave/cli/v3"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultConsoleAddr = "127.0.0.1:12137"
	defaultBackendAddr = "127.0.0.1:6112"
)

func ServeCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "serve",
		Description: "Start the backend and console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "console-addr",
				Value: defaultConsoleAddr,
				Usage: "Port for the console server",
			},
			&cli.StringFlag{
				Name:  "backend-addr",
				Value: defaultBackendAddr,
				Usage: "Port for the backend server",
			},
			&cli.StringFlag{
				Name:  "database-type",
				Value: "memory",
				Usage: "Database type (memory, sqlite)",
			},
			&cli.StringFlag{
				Name:  "sqlite-addr",
				Value: "dispel-multi-db.sqlite",
				Usage: "Path to sqlite database file",
			},
		},
	}

	cmd.Action = func(c *cli.Context) error {
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")

		// TODO: Use database-type flag and choose the database
		// db := memory.NewMemory()
		// db, err := database.NewLocal("database.sqlite")
		db, err := database.NewMemory()
		if err != nil {
			return err
		}
		queries, err := db.Queries()
		if err != nil {
			return err
		}

		pwd, _ := bcrypt.GenerateFromPassword([]byte("test"), 14)
		user, err := queries.CreateUser(context.TODO(), database.CreateUserParams{
			Username: "test",
			Password: string(pwd),
		})
		if err != nil {
			return err
		}

		_, err = queries.CreateCharacter(context.TODO(), database.CreateCharacterParams{
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

		bd := backend.NewBackend(queries)
		con := console.NewConsole(queries, bd)

		return con.Serve(c.Context, consoleAddr, backendAddr)
	}

	return cmd
}
