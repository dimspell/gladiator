package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) AdminScreen(w fyne.Window) fyne.CanvasObject {
	consoleStatus := binding.NewString()
	if c.Console == nil {
		_ = consoleStatus.Set("Console is not running")
	} else {
		_ = consoleStatus.Set(fmt.Sprintf("Console is running at %s", c.Console.Addr))
	}

	consoleScreen := func() fyne.CanvasObject {
		return container.NewPadded(
			widget.NewLabelWithData(consoleStatus),
		)
	}

	// # Play
	// First create a first user (db.users.count == 0)
	// Create -> link to signup form

	playContainer := container.NewCenter(
		widget.NewButton("Play", func() {
			go func() {
				err := c.StartBackend(c.Console.Addr, "")
				dialog.ShowError(err, w)
			}()
		}),
	)

	playTab := container.NewTabItemWithIcon("Play", theme.MediaPlayIcon(), playContainer)

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Status", theme.HomeIcon(), consoleScreen()),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), widget.NewLabel("Settings")),
		playTab,
	)

	tabs.SetTabLocation(container.TabLocationTop)

	headerText := "## Admin Panel"
	header := widget.NewRichTextFromMarkdown(headerText)

	stopCallback := func(b bool) {
		if b {
			log.Println("Welcome")
			w.SetContent(c.WelcomeScreen(w))
		}
	}

	return container.NewStack(
		container.NewBorder(
			container.NewPadded(
				container.NewHBox(
					header,
					layout.NewSpacer(),
					widget.NewButtonWithIcon("Stop", theme.LogoutIcon(), func() {
						cnf := dialog.NewConfirm(
							"Stopping server",
							"Do you want to stop the auth server\n and go back to the start page?\n",
							stopCallback,
							w)
						cnf.SetDismissText("Cancel")
						cnf.SetConfirmText("Yes, stop the server")
						cnf.Show()
					}),
				)),
			nil,
			nil,
			nil,
			tabs,
		),
	)
}
