package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dispel-re/dispel-multi/model"
)

type AdminScreenInputParams struct {
	DatabaseType string
	DatabasePath string
	BindAddress  string
}

func (c *Controller) AdminScreen(w fyne.Window, params *AdminScreenInputParams) fyne.CanvasObject {
	consoleScreen := func() fyne.CanvasObject {
		return container.NewVScroll(container.NewPadded(
			container.NewVBox(
				container.NewPadded(
					container.New(layout.NewFormLayout(),
						widget.NewLabelWithStyle("Run Mode:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
						widget.NewLabel(model.RunModeLAN),
						widget.NewLabelWithStyle("Bind Address:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
						widget.NewLabel(params.BindAddress),
						widget.NewLabelWithStyle("Database Type:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
						widget.NewLabel(params.DatabaseType),
						widget.NewLabelWithStyle("Database Path:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
						widget.NewLabel(params.DatabasePath),
					),
				),
			),
		))
	}

	onGoBack := func() {
		if c.Console != nil {
			cnf := dialog.NewConfirm(
				"Stopping server",
				"Do you want to stop the auth server\n and go back to the start page?\n",
				func(stop bool) {
					if !stop {
						return
					}

					c.StopBackend()

					if c.Console != nil {
						if err := c.StopConsole(); err != nil {
							dialog.ShowError(err, w)
						}
					}

					changePage(w, "Welcome", c.WelcomeScreen(w))
				}, w)
			cnf.SetDismissText("Cancel")
			cnf.SetConfirmText("Yes, stop the server")
			cnf.Show()
			return
		}
		changePage(w, "Welcome", c.WelcomeScreen(w))
	}

	consoleRunningLabel := binding.NewString()
	consoleRunningCheck := widget.NewLabelWithData(consoleRunningLabel)
	consoleRunningCheck.Alignment = fyne.TextAlignCenter
	c.consoleRunning.AddListener(binding.NewDataListener(func() {
		if _, isRunning := c.consoleProbe.Get(); isRunning {
			// consoleStart.Disable()
			// consoleStop.Enable()
			// createUser.Enable()
			consoleRunningLabel.Set("Console: Running")
			consoleRunningCheck.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			// consoleStart.Enable()
			// consoleStop.Disable()
			// createUser.Disable()
			consoleRunningLabel.Set("Console: Not Running")
			consoleRunningCheck.TextStyle = fyne.TextStyle{Bold: false}
		}
	}))

	return container.NewStack(
		container.NewBorder(
			container.NewPadded(
				container.NewHBox(
					widget.NewRichTextFromMarkdown("## Admin Panel"),
					layout.NewSpacer(),
					consoleRunningCheck,
					widget.NewButtonWithIcon("Stop", theme.LogoutIcon(), onGoBack),
				)),
			nil,
			nil,
			nil,
			container.NewAppTabs(
				container.NewTabItemWithIcon("Host", theme.HomeIcon(), consoleScreen()),
				container.NewTabItemWithIcon("Play", theme.MediaPlayIcon(), c.playView(w, params.BindAddress)),
			),
		),
	)
}
