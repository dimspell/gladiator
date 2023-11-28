package console

import (
	"context"

	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console/server"
	"github.com/dispel-re/dispel-multi/internal/database"
)

type Console struct {
	DB      *database.Queries
	Backend *backend.Backend
}

func NewConsole(db *database.Queries, b *backend.Backend) *Console {
	return &Console{
		DB:      db,
		Backend: b,
	}
}

func (c *Console) Serve(ctx context.Context, consoleAddr, backendAddr string) error {
	// "github.com/pocketbase/pocketbase"
	// app := pocketbase.NewWithConfig(pocketbase.Config{
	// 	DefaultDebug:         true,
	// 	DefaultDataDir:       "./pb_data.ignore",
	// 	DefaultEncryptionEnv: "",
	// 	HideStartBanner:      false,
	// 	DataMaxOpenConns:     core.DefaultDataMaxOpenConns,
	// 	DataMaxIdleConns:     core.DefaultDataMaxIdleConns,
	// 	LogsMaxOpenConns:     core.DefaultLogsMaxOpenConns,
	// 	LogsMaxIdleConns:     core.DefaultLogsMaxIdleConns,
	// })
	// return app.Start()

	consoleServer := server.ConsoleServer{
		DB:      c.DB,
		Backend: c.Backend,
	}
	return consoleServer.Serve(ctx, consoleAddr, backendAddr)
}
