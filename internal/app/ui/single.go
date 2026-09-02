package ui

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dimspell/gladiator/internal/app/ui/registrypatch"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/model"
)

type SinglePlayerScreenParameters struct {
	DatabaseType HostDatabaseTypeLabel
}

func (c *Controller) SinglePlayerScreen(w fyne.Window, initial *SinglePlayerScreenParameters) fyne.CanvasObject {
	const headerText = "Single Player"
	consoleAddr := "127.0.0.1:2137"

	pathLabel := widget.NewLabelWithStyle("Database Path:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	pathEntry := widget.NewEntry()
	{
		dir, _ := defaultDirectory()
		pathEntry.SetText(dir)
	}
	pathSelection := widget.NewButtonWithIcon("Select folder", theme.FolderOpenIcon(), selectDatabasePath(w, pathEntry))
	pathContainer := container.NewBorder(nil, nil, nil, pathSelection, pathEntry)

	comboGroup := widget.NewSelect(Values(databaseTypeText), func(value string) {
		if value == databaseTypeText[HostDatabaseTypeMemory] {
			pathLabel.Hide()
			pathContainer.Hide()
		} else {
			pathLabel.Show()
			pathContainer.Show()
		}
	})
	comboGroup.SetSelected(databaseTypeText[initial.DatabaseType])
	if initial.DatabaseType == HostDatabaseTypeMemory {
		pathLabel.Hide()
		pathContainer.Hide()
	}

	consoleRunningLabel := binding.NewString()
	consoleRunningCheck := widget.NewLabelWithData(consoleRunningLabel)
	consoleRunningCheck.Alignment = fyne.TextAlignCenter
	consoleStart := widget.NewButtonWithIcon("Start console", theme.MediaPlayIcon(), func() {
		databasePath := pathEntry.Text
		if err := os.MkdirAll(path.Dir(databasePath), 0755); err != nil {
			if !errors.Is(err, os.ErrExist) {
				dialog.ShowError(err, w)
				return
			}
		}

		databaseType, ok := databaseTypes[comboGroup.Selected]
		if !ok {
			dialog.ShowError(fmt.Errorf("unknown database type: %q", databaseType), w)
			return
		}
		if err := c.StartConsole(databaseType, databasePath, consoleAddr, model.RunModeSinglePlayer); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	consoleStop := widget.NewButtonWithIcon("Stop console", theme.MediaStopIcon(), func() {
		if err := c.StopConsole(); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})

	backendRunningLabel := binding.NewString()
	backendRunningCheck := widget.NewLabelWithData(backendRunningLabel)
	backendRunningCheck.Alignment = fyne.TextAlignCenter
	backendStart := widget.NewButtonWithIcon("Start backend", theme.MediaPlayIcon(), func() {
		if err := c.StartBackend("http://"+consoleAddr, &direct.ProxyLAN{MyIPAddress: "127.0.0.1"}); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	backendStop := widget.NewButtonWithIcon("Stop backend", theme.MediaStopIcon(), func() {
		c.StopBackend()
	})

	createUser := widget.NewButtonWithIcon("Create New User", theme.AccountIcon(), func() {
		centered := container.NewCenter()
		d := dialog.NewCustomWithoutButtons("Create New User", centered, w)
		centered.Add(c.signUpForm("http://"+consoleAddr, d.Hide, func(username string) {
			d.Hide()

			c.app.SendNotification(
				fyne.NewNotification("Created New User",
					fmt.Sprintf("You have successfully created a new user named %q.", username),
				))
		}, w))
		d.Show()
	})

	c.backendRunning.AddListener(binding.NewDataListener(func() {
		if _, isRunning := c.backendProbe.Status(); isRunning {
			backendStart.Disable()
			backendStop.Enable()
			_ = backendRunningLabel.Set("Backend: Running")
			backendRunningCheck.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			backendStart.Enable()
			backendStop.Disable()
			_ = backendRunningLabel.Set("Backend: Not Running")
			backendRunningCheck.TextStyle = fyne.TextStyle{Bold: false}
		}
	}))

	c.consoleRunning.AddListener(binding.NewDataListener(func() {
		if _, isRunning := c.consoleProbe.Status(); isRunning {
			consoleStart.Disable()
			consoleStop.Enable()
			createUser.Enable()
			_ = consoleRunningLabel.Set("Console: Running")
			consoleRunningCheck.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			consoleStart.Enable()
			consoleStop.Disable()
			createUser.Disable()
			_ = consoleRunningLabel.Set("Console: Not Running")
			consoleRunningCheck.TextStyle = fyne.TextStyle{Bold: false}
		}
	}))

	return container.NewBorder(
		container.NewPadded(
			headerContainer(headerText, func() {
				// TODO: It should be asked only whether the servers are running
				dialog.ShowConfirm("Are you sure?",
					"This action will close all servers if you have any started?",
					func(b bool) {
						if !b {
							return
						}

						c.StopBackend()

						if err := c.StopConsole(); err != nil {
							dialog.ShowError(err, w)
							return
						}

						changePage(w, "Start", c.StartScreen(w, startOptionPlay))
					}, w)
			}),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(
			container.NewPadded(
				container.NewVBox(
					renderRegistryNotes(),
					renderRegistryPatchContainer(w),
					renderStartServersNotes(),
					container.NewGridWithColumns(3,
						consoleRunningCheck,
						consoleStart,
						consoleStop,
						backendRunningCheck,
						backendStart,
						backendStop,
					),
					widget.NewAccordion(
						widget.NewAccordionItem("Advanced", container.NewVBox(
							container.New(
								layout.NewFormLayout(),
								widget.NewLabelWithStyle("Database Type:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
								comboGroup,
								pathLabel,
								pathContainer,
							),
						)),
					),
					renderCreateUserNotes(),
					container.NewHBox(
						layout.NewSpacer(),
						createUser,
						layout.NewSpacer(),
					),
					renderStartGameNotes(),
				),
			),
		),
	)
}

func renderRegistryNotes() *widget.RichText {
	registryUpdatedText := widget.NewRichTextFromMarkdown("**1. Point the installed game at this server (optional, reversible)**\n\n" +
		"This sets the game's server to `localhost` and its version to `1.30`. " +
		"The old values are saved first, so Restore puts them back.")
	registryUpdatedText.Wrapping = fyne.TextWrapWord

	return registryUpdatedText
}

func renderRegistryPatchContainer(w fyne.Window) fyne.CanvasObject {
	registryValueBinding := binding.NewString()
	refresh := func() {
		s, err := registrypatch.ReadServer()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if s == "" {
			s = "<unknown>"
		}
		_ = registryValueBinding.Set(fmt.Sprintf("Value: %q", s))
	}

	s, _ := registrypatch.ReadServer()
	if s == "" {
		s = "<unknown>"
	}
	_ = registryValueBinding.Set(fmt.Sprintf("Value: %q", s))

	checkButton := widget.NewButton("Check", func() {
		refresh()
	})

	patchButton := widget.NewButton("Use localhost", func() {
		before, _ := registrypatch.ReadServer()
		dialog.ShowConfirm("Point the game at this server?",
			fmt.Sprintf("Set the game's server from %q to %q and its version to %q? The old values are saved and can be restored.", before, "localhost", "1.30"),
			func(confirmed bool) {
				if !confirmed {
					return
				}
				if !registrypatch.PatchRegistry() {
					dialog.ShowError(fmt.Errorf("cannot change the setting (was the elevation prompt cancelled?)"), w)
					return
				}

				time.Sleep(1 * time.Second)
				refresh()
			}, w)
	})

	restoreButton := widget.NewButton("Restore", func() {
		if !registrypatch.RestoreRegistry() {
			dialog.ShowError(fmt.Errorf("nothing to restore (is there a saved value?)"), w)
			return
		}
		time.Sleep(1 * time.Second)
		refresh()
	})

	statusLabel := widget.NewLabelWithData(registryValueBinding)
	statusLabel.Alignment = fyne.TextAlignCenter

	return container.NewVBox(
		statusLabel,
		container.NewGridWithColumns(3,
			checkButton,
			patchButton,
			restoreButton,
		),
	)
}

func renderStartServersNotes() *widget.RichText {
	serversRunningText := widget.NewRichTextFromMarkdown("**2. Start the console and backend servers?**\n\n" +
		"You must have both servers running on your computer before starting the game. " +
		"Click on the buttons to start them:")
	serversRunningText.Wrapping = fyne.TextWrapWord

	return serversRunningText
}

func renderCreateUserNotes() *widget.RichText {
	createUserText := widget.NewRichTextFromMarkdown("**3. (Optional): Create new user**\n\n" +
		"In the game interface, you will be asked to sign in. " +
		"If you wish to create a brand new hero here, in the launcher interface, then please click on the Create New User button below.")
	createUserText.Wrapping = fyne.TextWrapWord

	return createUserText
}

func renderStartGameNotes() *widget.RichText {
	startGameText := widget.NewRichTextFromMarkdown("**4. Start game**\n\n" +
		"Start the game from the shortcut on your desktop or the Menu Start.")
	startGameText.Wrapping = fyne.TextWrapWord

	return startGameText
}
