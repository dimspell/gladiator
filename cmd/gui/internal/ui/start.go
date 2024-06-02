package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func StartScreen(w fyne.Window) fyne.CanvasObject {
	const headerText = "Start"

	radioOptions := []string{
		"Join - I want to join an already existing server.",
		"Host - I would like to host my own server over LAN.",
	}
	radioGroup := widget.NewRadioGroup(radioOptions, func(value string) {
		log.Println("Radio set to", value)
	})
	radioGroup.SetSelected(radioOptions[1])
	radioGroup.Required = true

	return container.NewPadded(container.NewVBox(
		container.New(
			layout.NewHBoxLayout(),
			widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
				log.Println("Welcome")
				w.SetContent(WelcomeScreen(w))
			}),
			widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		widget.NewLabel(""),

		widget.NewLabelWithStyle("What do you want to do?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		radioGroup,

		widget.NewLabel(""),
		container.NewCenter(
			widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
				log.Println(radioGroup.Selected)
				if radioGroup.Selected == radioOptions[0] {
					w.SetContent(JoinOptionsScreen(w))
					return
				}
				if radioGroup.Selected == radioOptions[1] {
					w.SetContent(HostScreen(w))
					return
				}
			}),
		),
	))
}
