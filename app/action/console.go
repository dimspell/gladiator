package action

import (
	"context"
	"fmt"

	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
	"github.com/urfave/cli/v3"
)

func ConsoleCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "console",
		Description: "Start console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "console-addr",
				Value: defaultConsoleAddr,
				Usage: "Port for the console server",
			},
			&cli.StringFlag{
				Name:  "database-type",
				Value: "memory",
				Usage: "Database type (memory, sqlite)",
			},
			&cli.StringFlag{
				Name:  "sqlite-path",
				Value: "dispel-multi.sqlite",
				Usage: "Path to sqlite database file",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		consoleAddr := cmd.String("console-addr")

		var (
			db  *database.SQLite
			err error
		)
		switch c.String("database-type") {
		case "memory":
			db, err = database.NewMemory()
			if err != nil {
				return err
			}
		case "sqlite":
			db, err = database.NewLocal(c.String("sqlite-path"))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown database type: %q", c.String("database-type"))
		}

		queries, err := db.Queries()
		if err != nil {
			return err
		}

		if err := database.Seed(queries); err != nil {
			return err
		}

		con := console.NewConsole(queries, nil)

		return con.Serve(ctx, consoleAddr, "")
	}

	return cmd
}
