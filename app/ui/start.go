package ui

import (
	"fmt"
	"log"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type startOption string

const (
	startOptionNone startOption = ""
	startOptionJoin startOption = "1_join"
	startOptionHost startOption = "2_host"
	startOptionPlay startOption = "3_play"
)

func (c *Controller) StartScreen(w fyne.Window, selectedOption startOption) fyne.CanvasObject {
	const headerText = "Start"

	radioOptions := map[startOption]string{
		startOptionJoin: "Join - I would like to join an existing server.",
		startOptionHost: "Host - I would like to host my own server via a LAN or WAN.",
		startOptionPlay: "Play alone - I want to play in single player mode.",
	}
	radioGroup := widget.NewRadioGroup(
		Values(radioOptions),
		func(value string) {
			slog.Debug(fmt.Sprintf("Radio set to %s", value), "page", "start")
		},
	)
	radioGroup.Required = true
	if selectedOption != startOptionNone {
		radioGroup.Selected = radioOptions[selectedOption]
	}

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
						if radioGroup.Selected == radioOptions[startOptionJoin] {
							w.SetContent(c.JoinOptionsScreen(w))
							return
						}
						if radioGroup.Selected == radioOptions[startOptionHost] {
							w.SetContent(c.HostScreen(w))
							return
						}
						if radioGroup.Selected == radioOptions[startOptionPlay] {
							w.SetContent(c.SinglePlayerScreen(w))
							return
						}
					}),
				),
			),
		),
	)
}
