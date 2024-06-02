package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) StartScreen(w fyne.Window) fyne.CanvasObject {
	const headerText = "Start"

	radioOptions := map[string]string{
		"join": "Join - I want to join an already existing server.",
		"host": "Host - I would like to host my own server over LAN.",
	}
	radioGroup := widget.NewRadioGroup(Values(radioOptions), func(value string) {
		log.Println("Radio set to", value)
	})
	radioGroup.Required = true

	return container.NewBorder(
		container.NewPadded(
			headerContainer(headerText, func() {
				log.Println("Welcome")
				w.SetContent(c.WelcomeScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewVBox(
				widget.NewLabel(""),

				widget.NewLabelWithStyle("What do you want to do?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				radioGroup,

				widget.NewLabel(""),
				container.NewCenter(
					widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
						log.Println(radioGroup.Selected)
						if radioGroup.Selected == radioOptions["join"] {
							w.SetContent(c.JoinOptionsScreen(w))
							return
						}
						if radioGroup.Selected == radioOptions["host"] {
							w.SetContent(c.HostScreen(w))
							return
						}
					}),
				),
			),
		),
	)
}
