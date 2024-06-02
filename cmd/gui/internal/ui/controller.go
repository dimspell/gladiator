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
}

func NewController(storage fyne.Storage) *Controller {
	return &Controller{}
}

func (c *Controller) ConsoleHandshake(consoleAddr string) error {
	client := &http.Client{Timeout: 10 * time.Second}

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

func (c *Controller) StartBackend(consoleAddr, backendAddr string) error {
	if backendAddr == "" {
		backendAddr = "127.0.0.1:6112"
	}

	bd := backend.NewBackend(backendAddr, consoleAddr)
	if err := bd.Start(context.TODO()); err != nil {
		return err
	}
	go bd.Listen()
	return nil
}
