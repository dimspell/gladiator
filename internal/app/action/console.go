package action

import (
	"context"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/urfave/cli/v3"
)

func ConsoleCommand(version string) *cli.Command {
	cmd := &cli.Command{
		Name:        "console",
		Description: "Start console server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "console-addr",
				Aliases: []string{"console-bind"},
				Value:   defaultConsoleAddr,
				Usage:   "Bind address for the console server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("CONSOLE_ADDR"), cli.EnvVar("CONSOLE_BIND")),
			},
			&cli.StringFlag{
				Name:    "console-public-addr",
				Value:   defaultConsoleAddr,
				Usage:   "Public address to the console server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("CONSOLE_PUBLIC_ADDR")),
			},
			&cli.StringFlag{
				Name:    "relay-addr",
				Aliases: []string{"relay-bind"},
				Usage:   "Bind address for the relay server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("RELAY_ADDR"), cli.EnvVar("RELAY_BIND")),
			},
			&cli.StringFlag{
				Name:    "relay-public-addr",
				Usage:   "Public address to the relay server",
				Sources: cli.NewValueSourceChain(cli.EnvVar("RELAY_PUBLIC_ADDR")),
			},
			&cli.StringFlag{
				Name:    "database-type",
				Value:   "memory",
				Usage:   "Database type (memory, sqlite)",
				Sources: cli.NewValueSourceChain(cli.EnvVar("DATABASE_TYPE")),
			},
			&cli.StringFlag{
				Name:    "sqlite-path",
				Value:   "dispel-multi.sqlite",
				Usage:   "Path to sqlite database file",
				Sources: cli.NewValueSourceChain(cli.EnvVar("SQLITE_PATH")),
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

		co, err := selectConsoleOptions(c, version)
		if err != nil {
			return err
		}
		con := console.NewConsole(db, co...)

		start, stop := con.Handlers()
		return con.Graceful(ctx, start, stop)
	}

	return cmd
}
