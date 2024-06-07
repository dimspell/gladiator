package ui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console"
)

type Controller struct {
	Console *console.Console
	Backend *backend.Backend

	consoleProbe chan bool
	backendProbe chan bool
}

func NewController(storage fyne.Storage) *Controller {
	return &Controller{
		consoleProbe: make(chan bool),
		backendProbe: make(chan bool),
	}
}

func (c *Controller) ConsoleHandshake(consoleAddr string) error {
	client := &http.Client{Timeout: 3 * time.Second}

	res, err := client.Get(fmt.Sprintf("http://%s/.well-known/dispel-multi.json", consoleAddr))
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
	c.Backend = backend.NewBackend("127.0.0.1:6112", consoleAddr)
	if err := c.Backend.Start(context.TODO()); err != nil {
		return err
	}
	go c.Backend.Listen()

	go func() {
		// Delay
		ticker := time.NewTicker(time.Second * 2)

		for {
			select {
			case <-ticker.C:
				// c.Backend.Status
			}
		}
	}()
	return nil
}

func (c *Controller) StopBackend() {
	if c.Backend == nil {
		return
	}
	c.Backend.Shutdown()
}
