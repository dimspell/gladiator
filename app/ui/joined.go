package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) JoinedScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewBorder(
		container.NewPadded(
			headerContainer("Sign-up", func() {
				log.Println("Join")
				w.SetContent(c.JoinOptionsScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		container.NewPadded(
			widget.NewButtonWithIcon("Stop backend", theme.HomeIcon(), func() {
				c.StopBackend()
				w.SetContent(c.WelcomeScreen(w))
			}),
		),
	)
}
