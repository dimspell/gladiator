package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) JoinDefineScreen(w fyne.Window) fyne.CanvasObject {
	str := binding.NewString()

	label1 := widget.NewLabel("Authorization Server Address:")
	value1 := widget.NewEntryWithData(str)
	value1.Validator = nil
	value1.PlaceHolder = "Full address, for example http://127.0.0.1:2137"

	headerText := "Join a server - define auth server"

	return container.NewBorder(
		container.NewPadded(headerContainer(headerText, func() {
			log.Println("Welcome")
			w.SetContent(c.JoinOptionsScreen(w))
		})),
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewVBox(
				widget.NewLabel(""),

				label1,
				value1,

				widget.NewLabel(""),
				container.NewCenter(
					widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
						log.Println(str.Get())
					}),
				),
			),
		),
	)
}
