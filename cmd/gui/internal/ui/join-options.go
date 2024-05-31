package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func JoinOptionsScreen(w fyne.Window) fyne.CanvasObject {
	headerText := "Join a server"

	var radioValue string

	radioOptions := []string{
		"Use dispelmulti.net network",
		"Use loopback for testing (127.0.0.1:2137)",
		"Define your own",
	}
	radioGroup := widget.NewRadioGroup(radioOptions, func(value string) {
		log.Println("Radio set to", value)
		radioValue = value
	})

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

		widget.NewLabel("Authorization Server Address:"),
		radioGroup,

		widget.NewLabel(""),
		container.NewCenter(
			widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
				log.Println(radioValue)
				if radioValue == radioOptions[1] {
					// Start backend (popup?)
					w.SetContent(SignInScreen(w))
					return
				}
				if radioValue == radioOptions[2] {
					w.SetContent(JoinCustomScreen(w))
					return
				}
			}),
		),
	))
}
