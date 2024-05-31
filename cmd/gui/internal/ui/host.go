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
	label1 := widget.NewLabel("Bind Address:")
	value1 := widget.NewEntry()
	label2 := widget.NewLabel("Database Type:")
	value2 := widget.NewEntry()
	label3 := widget.NewLabel("Database Path:")
	value3 := widget.NewEntry()

	formGrid := container.New(layout.NewFormLayout(), label1, value1, label2, value2, label3, value3)

	headerText := "Host a server"
	header := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewPadded(container.NewVBox(
		container.New(
			layout.NewHBoxLayout(),
			widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
				log.Println("Welcome")
				w.SetContent(WelcomeScreen(w))
			}),
			header,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),

		formGrid,
		container.NewCenter(
			widget.NewButtonWithIcon("Submit", theme.NavigateNextIcon(), func() {
				w.SetContent(AdminScreen(w))
			}),
		),
	))
}
