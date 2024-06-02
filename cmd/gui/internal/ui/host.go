package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) HostScreen(w fyne.Window) fyne.CanvasObject {
	pathLabel := widget.NewLabel("Database Path:")
	pathEntry := widget.NewEntry()

	comboOptions := []string{
		"Saved on disk (sqlite)",
		"Stored in-memory (for testing)",
	}
	comboGroup := widget.NewSelect(comboOptions, func(value string) {
		log.Println("Select set to", value)

		pathNotUsed := value == comboOptions[1]
		pathLabel.Hidden = pathNotUsed
		pathEntry.Hidden = pathNotUsed
	})

	bindLabel := widget.NewLabel("Bind Address:")
	bindEntry := widget.NewEntry()
	bindEntry.PlaceHolder = "Example: 0.0.0.0:2137"
	bindEntry.Text = "127.0.0.1:2137"

	typeLabel := widget.NewLabel("Database Type:")
	typeEntry := comboGroup

	comboGroup.SetSelected(comboOptions[1])
	pathLabel.Hidden = true
	pathEntry.Hidden = true

	formGrid := container.New(layout.NewFormLayout(), bindLabel, bindEntry, typeLabel, typeEntry, pathLabel, pathEntry)

	headerText := "Host a server"
	header := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewPadded(container.NewVBox(
		container.New(
			layout.NewHBoxLayout(),
			widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
				log.Println("Start")
				w.SetContent(c.StartScreen(w))
			}),
			header,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),

		formGrid,

		widget.NewLabel(""),
		container.NewCenter(
			widget.NewButtonWithIcon("Submit", theme.NavigateNextIcon(), func() {
				w.SetContent(c.AdminScreen(w))
			}),
		),
	))
}
