package ui

import (
	"fyne.io/fyne/v2"
)

type Controller struct {
}

func NewController(storage fyne.Storage) *Controller {
	return &Controller{}
}
