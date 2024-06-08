package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/dispel-re/dispel-multi/app/ui/probe"
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/backend/packetlogger"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
)

type Controller struct {
	myIPAddress string
	Console     *console.Console
	Backend     *backend.Backend

	consoleStop console.GracefulFunc

	app          fyne.App
	consoleProbe *probe.Probe
	backendProbe *probe.Probe

	backendRunning binding.Bool
	consoleRunning binding.Bool
}

func NewController(fyneApp fyne.App) *Controller {
	return &Controller{
		// TODO: Define the IP address used for proxy
		myIPAddress: "127.0.0.1",

		app:            fyneApp,
		consoleProbe:   probe.NewProbe(),
		backendProbe:   probe.NewProbe(),
		backendRunning: binding.NewBool(),
		consoleRunning: binding.NewBool(),
	}
}

func (c *Controller) StartConsole(databaseType, databasePath, consoleAddr string) error {
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

	queries, err := db.Queries()
	if err != nil {
		return err
	}

	// Update the database to the latest migration
	if err := database.Seed(queries); err != nil {
		slog.Warn("Seed queries failed", "error", err)
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

	c.Console = console.NewConsole(db, queries, consoleAddr)
	start, stop := c.Console.Handlers()
	c.consoleStop = func(ctx context.Context) error {
		if c.Console == nil {
			return nil
		}
		c.consoleProbe.StopStartupProbe()
		err := stop(ctx)
		c.Console = nil
		cancel()
		return err
	}

	c.consoleProbe.StartupProbe(probe.HttpGet(
		fmt.Sprintf("http://%s/_health", consoleAddr),
	))
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

func (c *Controller) ConsoleHandshake(consoleAddr string) error {
	client := &http.Client{Timeout: 3 * time.Second}

	if !strings.Contains(consoleAddr, "://") {
		consoleAddr = "http://" + consoleAddr
	}
	res, err := client.Get(fmt.Sprintf("%s/.well-known/dispel-multi.json", consoleAddr))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("incorrect http-status code: %d", res.StatusCode)
	}

	// TODO: Read configuration parameters
	// var resp struct{}
	// if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// log.Println(resp)
	return nil
}

func (c *Controller) StartBackend(consoleAddr string) error {
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

	c.Backend = backend.NewBackend("127.0.0.1:6112", consoleAddr, c.myIPAddress)
	if err := c.Backend.Start(); err != nil {
		cancel()
		return err
	}

	packetsLogFile, err := os.CreateTemp("", "dispel-multi-packets.txt")
	if err == nil {
		c.Backend.PacketLogger = slog.New(packetlogger.New(packetsLogFile, &packetlogger.Options{
			Level: slog.LevelDebug,
		}))
	}

	// Healthcheck
	c.backendProbe.Signal(probe.StatusRunning)

	// Start listening
	go func() {
		c.Backend.Listen()
		c.backendProbe.Signal(probe.StatusNotRunning)
		cancel()
		if packetsLogFile != nil {
			if err := packetsLogFile.Close(); err != nil {
				slog.Warn("Failed to close packet log file", "error", err)
			}
		}
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
