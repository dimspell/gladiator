package action

import (
	"context"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/console"
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
		db, err := selectDatabaseType(c)
		if err != nil {
			return err
		}
		defer func() {
			if err := db.Close(); err != nil {
				slog.Error("Failed to close database", logging.Error(err))
			}
		}()

		co, err := selectConsoleOptions(c)
		if err != nil {
			return err
		}
		con := console.NewConsole(db, co...)

		start, stop := con.Handlers()
		return con.Graceful(ctx, start, stop)
	}

	return cmd
}
