package action

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/console"
	"github.com/dimspell/gladiator/console/database"
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

		if err := database.Seed(db.Write); err != nil {
			slog.Warn("Seed queries failed", "error", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close database", "error", err)
			}
		}()

		con := console.NewConsole(db, consoleAddr)
		start, stop := con.Handlers()
		return con.Graceful(ctx, start, stop)
	}

	return cmd
}
