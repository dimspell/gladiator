package ui

import (
	"fmt"

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
			loadingDialog := dialog.NewCustomWithoutButtons("Connecting to auth server...", widget.NewProgressBarInfinite(), w)
			loadingDialog.Show()

			if err := c.StartBackend(c.Console.Addr); err != nil {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}

			loadingDialog.Hide()
			changePage(w, "Joined", c.JoinedScreen(w))
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

	stopCallback := func(stop bool) {
		if !stop {
			return
		}

		if c.Console != nil {
			if err := c.StopConsole(); err != nil {
				dialog.ShowError(err, w)
			}
		}

		changePage(w, "Welcome", c.WelcomeScreen(w))
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
