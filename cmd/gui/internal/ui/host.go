package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func HostScreen(w fyne.Window) fyne.CanvasObject {
	bindLabel := widget.NewLabel("Bind Address:")
	bindEntry := widget.NewEntry()
	typeLabel := widget.NewLabel("Database Type:")
	typeEntry := widget.NewEntry()
	pathLabel := widget.NewLabel("Database Path:")
	pathEntry := widget.NewEntry()

	formGrid := container.New(layout.NewFormLayout(), bindLabel, bindEntry, typeLabel, typeEntry, pathLabel, pathEntry)

	headerText := "Host a server"
	header := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewPadded(container.NewVBox(
		container.New(
			layout.NewHBoxLayout(),
			widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
				log.Println("Start")
				w.SetContent(StartScreen(w))
			}),
			header,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),

		formGrid,

		widget.NewLabel(""),
		container.NewCenter(
			widget.NewButtonWithIcon("Submit", theme.NavigateNextIcon(), func() {
				w.SetContent(AdminScreen(w))
			}),
		),
	))
}
