package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type AdminScreenInputParams struct {
	DatabaseType string
	DatabasePath string
	BindAddress  string
}

func (c *Controller) AdminScreen(w fyne.Window, params *AdminScreenInputParams) fyne.CanvasObject {
	consoleScreen := func() fyne.CanvasObject {
		pages := map[widget.ListItemID]string{
			0: "Configuration",
		}
		list := widget.NewList(
			func() int {
				return len(pages)
			},
			func() fyne.CanvasObject {
				return widget.NewLabel("")
			},
			func(id widget.ListItemID, object fyne.CanvasObject) {
				p := pages[id]
				object.(*widget.Label).SetText(p)
			},
		)
		list.Select(0)

		var paramsContainer fyne.CanvasObject
		if c.Console != nil {
			formContainer := container.New(layout.NewFormLayout())
			paramsMap := map[string]string{
				"Run Mode":      c.Console.RunMode.String(),
				"Bind Address":  params.BindAddress,
				"Database Type": params.DatabaseType,
				"Database Path": params.DatabasePath,
			}
			for k, v := range paramsMap {
				formContainer.Add(widget.NewLabelWithStyle(k+":", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}))
				formContainer.Add(widget.NewLabel(v))
			}

			paramsContainer = container.NewVBox(container.NewPadded(formContainer))
		} else {
			paramsContainer = container.NewCenter(
				widget.NewLabel("The console is not running"),
			)
		}

		split := container.NewHSplit(
			list,
			container.NewVScroll(container.NewPadded(
				paramsContainer,
			)),
		)
		split.Offset = 0.25

		return split
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
