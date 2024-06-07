package ui

import (
	"errors"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) SinglePlayerScreen(w fyne.Window) fyne.CanvasObject {
	const headerText = "Single Player"
	var (
		consoleAddrIP, consoleAddrPort = "127.0.0.1", "2137"
		// databaseType = "sqlite"
		databaseType = "memory"
	)

	consoleRunning := binding.NewBool()
	consoleRunningCheck := widget.NewCheckWithData("Console Running", consoleRunning)

	consoleStart := widget.NewButtonWithIcon("Start console", theme.HomeIcon(), func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		databasePath := homeDir + string(os.PathSeparator) + "dispel-multi"
		if err := os.Mkdir(databasePath, 0755); err != nil {
			if !errors.Is(err, os.ErrExist) {
				dialog.ShowError(err, w)
				return
			}
		}
		databasePath += string(os.PathSeparator)
		databasePath += "dispel-multi.sql"

		if err := c.StartConsole(databaseType, databasePath, consoleAddrIP, consoleAddrPort); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})

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
				consoleRunningCheck,
				consoleStart,
			),
		),
	)
}
