package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/probe"
)

type Controller struct {
	Console *console.Console
	Backend *backend.Backend

	consoleStop console.GracefulFunc

	app          fyne.App
	version      string
	consoleProbe *probe.Probe
	backendProbe *probe.Probe

	backendRunning binding.Bool
	consoleRunning binding.Bool
}

func NewController(fyneApp fyne.App, version string) *Controller {
	return &Controller{
		app:     fyneApp,
		version: version,

		consoleProbe:   probe.NewProbe(),
		backendProbe:   probe.NewProbe(),
		backendRunning: binding.NewBool(),
		consoleRunning: binding.NewBool(),
	}
}

func (c *Controller) StartConsole(databaseType, databasePath, consoleAddr string, runMode model.RunMode) error {
	if c.Console != nil {
		slog.Warn("Console is already running")
		return nil
	}

	slog.Info("Starting the console server",
		"databaseType", databaseType,
		"consoleAddr", consoleAddr,
	)

	// Configure the database connection
	var (
		db  *database.SQLite
		err error
	)
	switch databaseType {
	case "sqlite":
		db, err = database.NewLocal(databasePath)
		if err != nil {
			return err
		}
	case "memory":
		db, err = database.NewMemory()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown database type")
	}

	// Update the database to the latest migration
	if err := database.Seed(db.Write); err != nil {
		slog.Warn("Seed queries failed, likely it was run already", logging.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				continue
			case code := <-c.consoleProbe.SignalChange:
				if code == probe.StatusRunning {
					_ = c.consoleRunning.Set(true)
				} else {
					_ = c.consoleRunning.Set(false)
				}
			}
		}
	}()

	c.Console = console.NewConsole(db, console.WithConsoleAddr(consoleAddr, "http://"+consoleAddr))
	c.Console.Config.RunMode = runMode

	start, stop := c.Console.Handlers()
	c.consoleStop = func(ctx context.Context) error {
		if c.Console == nil {
			return nil
		}
		c.consoleProbe.StopStartupProbe()
		err := stop(ctx)
		db.Close()
		c.Console = nil
		cancel()
		return err
	}

	c.consoleProbe.Check(probe.NewHTTPHealthChecker(fmt.Sprintf("http://%s/_health", consoleAddr)).Check)
	go func() {
		if err := start(ctx); err != nil {
			cancel()
			return
		}
	}()
	return nil
}

func (c *Controller) StopConsole() error {
	slog.Info("Going to stop the console server")

	if c.Console == nil {
		slog.Warn("Console has been already shut down")
		return nil
	}
	if err := c.consoleStop(context.TODO()); err != nil {
		return err
	}

	return nil
}

func (c *Controller) StartBackend(consoleAddr string, proxy backend.Proxy) error {
	if c.Backend != nil {
		slog.Warn("Backend is already running")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-ctx.Done():
				continue
			case code := <-c.backendProbe.SignalChange:
				if code == probe.StatusRunning {
					_ = c.backendRunning.Set(true)
				} else {
					_ = c.backendRunning.Set(false)
				}
			}
		}
	}()

	c.Backend = backend.NewBackend("127.0.0.1:6112", consoleAddr, proxy)
	if err := c.Backend.Start(); err != nil {
		cancel()
		return err
	}
	c.Backend.SignalServerURL = fmt.Sprintf("%s/lobby", consoleAddr)

	// Healthcheck
	c.backendProbe.Signal(probe.StatusRunning)

	// Start listening
	go func() {
		c.Backend.Listen()
		c.backendProbe.Signal(probe.StatusNotRunning)
		cancel()
	}()

	return nil
}

func (c *Controller) StopBackend() {
	slog.Info("Going to stop the backend server")
	if c.Backend == nil {
		slog.Warn("Backend has been already shut down")
		return
	}
	c.backendProbe.Signal(probe.StatusClosing)

	c.Backend.Shutdown()
	c.Backend = nil

	c.backendProbe.Signal(probe.StatusNotRunning)
}

func (c *Controller) ConsoleRunning() bool {
	return c.Console != nil
}
