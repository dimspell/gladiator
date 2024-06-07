package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func (c *Controller) SinglePlayerScreen(w fyne.Window) fyne.CanvasObject {
	const headerText = "Single Player"

	return container.NewBorder(
		container.NewPadded(
			headerContainer(headerText, func() {
				log.Println("Welcome")
				w.SetContent(c.StartScreen(w, startOptionPlay))
			}),
		),
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewVBox(
			//
			),
		),
	)
}
