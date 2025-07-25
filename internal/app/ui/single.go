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
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/backend/registrypatch"
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
		if err := c.StartBackend("http://"+consoleAddr, &direct.ProxyLAN{"127.0.0.1"}); err != nil {
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
			backendRunningLabel.Set("Backend: Running")
			backendRunningCheck.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			backendStart.Enable()
			backendStop.Disable()
			backendRunningLabel.Set("Backend: Not Running")
			backendRunningCheck.TextStyle = fyne.TextStyle{Bold: false}
		}
	}))

	c.consoleRunning.AddListener(binding.NewDataListener(func() {
		if _, isRunning := c.consoleProbe.Status(); isRunning {
			consoleStart.Disable()
			consoleStop.Enable()
			createUser.Enable()
			consoleRunningLabel.Set("Console: Running")
			consoleRunningCheck.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			consoleStart.Enable()
			consoleStop.Disable()
			createUser.Disable()
			consoleRunningLabel.Set("Console: Not Running")
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
	registryUpdatedText := widget.NewRichTextFromMarkdown("**1. Update the registry (e.g. with regedit)**\n\n" +
		"Make sure the value of `HKEY_LOCAL_MACHINE\\SOFTWARE\\WOW6432Node\\AbalonStudio\\Dispel\\Multi\\Server` key is set to `localhost`. " +
		"If not, then please change it.")
	registryUpdatedText.Wrapping = fyne.TextWrapWord

	return registryUpdatedText
}

func renderRegistryPatchContainer(w fyne.Window) fyne.CanvasObject {
	registryValueBinding := binding.NewString()
	changeRegistryValue := func(registryValue string) {
		if registryValue == "" {
			registryValue = "<unknown>"
		}
		registryValueBinding.Set(fmt.Sprintf("Value: %q", registryValue))
	}

	registryValue, _ := registrypatch.ReadServer()
	changeRegistryValue(registryValue)

	checkButton := widget.NewButton("Check registry", func() {
		s, err := registrypatch.ReadServer()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		changeRegistryValue(s)
		dialog.ShowInformation("Registry value", fmt.Sprintf("Current value: %q", s), w)
	})

	patchButton := widget.NewButton("Patch registry", func() {
		before, _ := registrypatch.ReadServer()

		if !registrypatch.PatchRegistry() {
			dialog.ShowError(fmt.Errorf("cannot change registry key"), w)
			return
		}

		time.Sleep(1 * time.Second)

		after, err := registrypatch.ReadServer()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		changeRegistryValue(after)
		dialog.ShowInformation("Changed Windows Registry", fmt.Sprintf("From %q to %q", before, after), w)
	})

	statusLabel := widget.NewLabelWithData(registryValueBinding)
	statusLabel.Alignment = fyne.TextAlignCenter

	return container.NewGridWithColumns(3,
		statusLabel,
		checkButton,
		patchButton,
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
		"Start the Dispel Multi game from the shortcut on your desktop or the Menu Start.")
	startGameText.Wrapping = fyne.TextWrapWord

	return startGameText
}
