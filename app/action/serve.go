package action

import (
	"context"
	"fmt"

	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
)

const (
	defaultConsoleAddr = "127.0.0.1:2137"
	defaultBackendAddr = "127.0.0.1:6112"
	defaultMyIPAddr    = "127.0.0.1"
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
				Name:  "my-ip-addr",
				Value: defaultMyIPAddr,
				Usage: "IP address used in intercommunication between the users",
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
		consoleAddr := c.String("console-addr")
		backendAddr := c.String("backend-addr")
		myIpAddr := c.String("my-ip-addr")

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

		// if err := database.Seed(queries); err != nil {
		// 	return err
		// }

		bd := backend.NewBackend(backendAddr, consoleAddr, myIpAddr)
		con := console.NewConsole(db, queries, consoleAddr)
		startConsole, stopConsole := con.Handlers()

		group, groupContext := errgroup.WithContext(ctx)
		group.Go(func() error {
			return startConsole(groupContext)
		})
		group.Go(func() error {
			if err := bd.Start(groupContext); err != nil {
				return err
			}
			bd.Listen()
			return nil
		})

		if err := group.Wait(); err != nil {
			bd.Shutdown()
			_ = stopConsole(context.TODO())
		}
		return nil
	}

	return cmd
}
