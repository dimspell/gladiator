package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dimspell/gladiator/internal/model"
)

type AdminScreenInputParams struct {
	DatabaseType string
	DatabasePath string
	BindAddress  string
}

func (c *Controller) AdminScreen(w fyne.Window, params *AdminScreenInputParams, metadata *model.WellKnown) fyne.CanvasObject {
	wrapConsoleRunning := func(children func() fyne.CanvasObject) fyne.CanvasObject {
		if !c.ConsoleRunning() {
			return container.NewCenter(
				widget.NewLabel("The console is not running"),
			)
		}
		return children()
	}

	configurationView := func() fyne.CanvasObject {
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

		return container.NewVBox(container.NewPadded(formContainer))
	}
	actionView := func() fyne.CanvasObject {
		return container.NewVBox(
			widget.NewLabel("Actions"),
			widget.NewButton("Delete all game rooms", func() {
				// if c.Console == nil {
				// 	dialog.ShowError(fmt.Errorf("The console is not running"), w)
				// 	return
				// }
				//
				// loadingDialog := dialog.NewCustomWithoutButtons("Deleting all games", widget.NewProgressBarInfinite(), w)
				// loadingDialog.Show()
				//
				// err := errors.Join(
				// 	func() error {
				// 		if err := c.Console.DB.Write.DeleteAllGameRoomPlayers(context.TODO()); err != nil {
				// 			return fmt.Errorf("could not delete all game room players: %w", err)
				// 		}
				// 		return nil
				// 	}(),
				// 	func() error {
				// 		if err := c.Console.DB.Write.DeleteAllGameRooms(context.TODO()); err != nil {
				// 			return fmt.Errorf("could not delete all game rooms: %w", err)
				// 		}
				// 		return nil
				// 	}(),
				// )
				//
				// loadingDialog.Hide()
				// if err != nil {
				// 	dialog.ShowError(err, w)
				// 	return
				// }
				dialog.ShowInformation("Not working", "Not working anymore", w)
			}),
		)
	}

	consoleScreen := func() fyne.CanvasObject {
		scrollPane := container.NewPadded()

		pages := map[widget.ListItemID]string{
			0: "Configuration",
			1: "Actions",
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
				go object.(*widget.Label).SetText(p)
			},
		)
		list.Select(0)
		list.OnUnselected = func(id widget.ListItemID) {
			scrollPane.RemoveAll()
		}
		list.OnSelected = func(id widget.ListItemID) {
			switch id {
			case 0:
				scrollPane.Add(wrapConsoleRunning(configurationView))
				break
			case 1:
				scrollPane.Add(wrapConsoleRunning(actionView))
				break
			}
		}

		scrollPane.Add(wrapConsoleRunning(configurationView))

		split := container.NewHSplit(list, container.NewVScroll(scrollPane))
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
		if _, isRunning := c.consoleProbe.Status(); isRunning {
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
				container.NewTabItemWithIcon("Play", theme.MediaPlayIcon(), c.playView(w, params.BindAddress, metadata)),
			),
		),
	)
}
