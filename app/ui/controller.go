package ui

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
)

type Controller struct {
	Console *console.Console
	Backend *backend.Backend

	consoleStop console.GracefulFunc

	app          fyne.App
	consoleProbe chan bool
	backendProbe chan bool
}

func NewController(fyneApp fyne.App) *Controller {
	return &Controller{
		app:          fyneApp,
		consoleProbe: make(chan bool),
		backendProbe: make(chan bool),
	}
}

func (c *Controller) ConsoleHandshake(consoleAddr string) error {
	client := &http.Client{Timeout: 3 * time.Second}

	// TODO: name the schema
	consoleSchema := "http"
	res, err := client.Get(fmt.Sprintf("%s://%s/.well-known/dispel-multi.json", consoleSchema, consoleAddr))
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
		c.Backend.Shutdown()
		c.Backend = nil
	}
	// TODO: Define the IP address used for proxy
	c.Backend = backend.NewBackend("127.0.0.1:6112", consoleAddr, "")
	if err := c.Backend.Start(context.TODO()); err != nil {
		return err
	}
	go c.Backend.Listen()

	// go func() {
	// 	// Delay
	// 	ticker := time.NewTicker(time.Second * 2)
	//
	// 	for {
	// 		if c.Backend == nil {
	// 			return
	// 		}
	//
	// 		select {
	// 		case <-ticker.C:
	// 			if c.Backend.Status == backend.StatusRunning {
	// 			}
	// 		}
	// 	}
	// }()
	return nil
}

func (c *Controller) StopBackend() {
	if c.Backend == nil {
		return
	}
	c.Backend.Shutdown()
}

func (c *Controller) StartConsole(databaseType, databasePath string, bindIP, bindPort string) error {
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
		return err
	}

	c.Console = console.NewConsole(queries, net.JoinHostPort(bindIP, bindPort))
	start, stop := c.Console.Handlers()
	c.consoleStop = stop

	go func() {
		if err := start(context.TODO()); err != nil {
			return
		}
	}()
	return nil
}

func (c *Controller) StopConsole() error {
	if c.consoleStop == nil {
		return nil
	}
	return c.consoleStop(context.TODO())
}
