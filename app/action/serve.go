package action

import (
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/internal/database/memory"
	"github.com/urfave/cli/v3"
)

const (
	defaultConsoleAddr = "0.0.0.0:2137"
	defaultBackendAddr = "0.0.0.0:6112"
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
		db := memory.NewMemory()
		bd := backend.NewBackend(db)
		con := console.NewConsole(db, bd)

		return con.Serve(c.Context, consoleAddr, backendAddr)
	}

	return cmd
}
