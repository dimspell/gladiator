package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func AdminScreen(w fyne.Window) fyne.CanvasObject {
	consoleScreen := func() fyne.CanvasObject {
		return widget.NewLabel("Console is running at 127.0.0.1:2137")
	}

	// # Play
	// First create a first user (db.users.count == 0)
	// Create -> link to signup form

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Status", theme.HomeIcon(), consoleScreen()),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), widget.NewLabel("Settings")),
		container.NewTabItemWithIcon("Play", theme.MediaPlayIcon(), widget.NewLabel("Play")),
	)

	// tabs.SetTabLocation(container.TabLocationLeading)

	headerText := "## Console Admin Panel"
	header := widget.NewRichTextFromMarkdown(headerText)

	stopCallback := func(b bool) {
		if b {
			log.Println("Welcome")
			w.SetContent(WelcomeScreen(w))
		}
	}

	return container.NewPadded(
		container.NewVBox(
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
			),
			tabs,
		),
	)
}
