package ui

import (
	"fyne.io/fyne/v2"
	"github.com/dispel-re/dispel-multi/console"
)

type Controller struct {
	Console *console.Console
}

func NewController(storage fyne.Storage) *Controller {
	return &Controller{}
}
