package ui

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dispel-re/dispel-multi/console/database"
)

type SinglePlayerScreenParameters struct {
	DatabaseType HostDatabaseType
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

	comboOptions := map[HostDatabaseType]string{
		HostDatabaseTypeSqlite: "Saved on disk (sqlite)",
		HostDatabaseTypeMemory: "Stored in-memory (for testing)",
	}
	databaseTypes := map[string]string{
		comboOptions[HostDatabaseTypeSqlite]: "sqlite",
		comboOptions[HostDatabaseTypeMemory]: "memory",
	}
	comboGroup := widget.NewSelect(Values(comboOptions), func(value string) {
		log.Println("Select set to", value)

		if value == comboOptions[HostDatabaseTypeMemory] {
			pathLabel.Hide()
			pathContainer.Hide()
		} else {
			pathLabel.Show()
			pathContainer.Show()
		}
	})
	comboGroup.SetSelected(comboOptions[initial.DatabaseType])
	if initial.DatabaseType == HostDatabaseTypeMemory {
		pathLabel.Hide()
		pathContainer.Hide()
	}

	advancedContainer := container.NewVBox(
		container.New(
			layout.NewFormLayout(),
			widget.NewLabelWithStyle("Database Type:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
			comboGroup,
			pathLabel,
			pathContainer,
		),
	)

	consoleRunningLabel := binding.NewString()
	consoleRunningCheck := widget.NewLabelWithData(consoleRunningLabel)
	consoleRunningCheck.Alignment = fyne.TextAlignCenter
	consoleStart := widget.NewButtonWithIcon("Start console", theme.MediaPlayIcon(), func() {
		dispelDir, err := defaultDirectory()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		if err := os.MkdirAll(path.Dir(dispelDir), 0755); err != nil {
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
		if err := c.StartConsole(databaseType, dispelDir, consoleAddr); err != nil {
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
		if err := c.StartBackend(consoleAddr); err != nil {
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
		centered.Add(c.signUpForm(d.Hide, func(user database.User) {
			d.Hide()

			c.app.SendNotification(
				fyne.NewNotification("Created New User",
					fmt.Sprintf("You have successfully created a new user named %q.", user.Username),
				))
		}, w))
		d.Show()
	})

	c.backendRunning.AddListener(binding.NewDataListener(func() {
		if _, isRunning := c.backendProbe.Get(); isRunning {
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
		if _, isRunning := c.consoleProbe.Get(); isRunning {
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

	registryUpdatedText := widget.NewRichTextFromMarkdown("**1. Update the registry (e.g. with regedit)**\n\n" +
		"Make sure the value of `HKEY_LOCAL_MACHINE\\SOFTWARE\\WOW6432Node\\AbalonStudio\\Dispel\\Multi\\Server` key is set to `localhost`. " +
		"If not, then please change it.")
	registryUpdatedText.Wrapping = fyne.TextWrapWord

	serversRunningText := widget.NewRichTextFromMarkdown("**2. Start the console and backend servers?**\n\n" +
		"You must have both servers running on your computer before starting the game. " +
		"Click on the buttons to start them:")
	serversRunningText.Wrapping = fyne.TextWrapWord

	createUserText := widget.NewRichTextFromMarkdown("**3. (Optional): Create new user**\n\n" +
		"In the game interface, you will be asked to sign in. " +
		"You can use `\"player\"` account with any password you want (e.g. \"test\"). " +
		"However, if you wish to create a brand new hero here, in the launcher interface, then please click on the Create New User button below.")
	createUserText.Wrapping = fyne.TextWrapWord

	startGameText := widget.NewRichTextFromMarkdown("**4. Start game**\n\n" +
		"Start the Dispel Multi game from the shortcut on your desktop or the Menu Start.")
	startGameText.Wrapping = fyne.TextWrapWord

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

						log.Println("Start")
						w.SetContent(c.StartScreen(w, startOptionPlay))
					}, w)
			}),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(
			container.NewPadded(
				container.NewVBox(
					registryUpdatedText,
					serversRunningText,
					container.NewGridWithColumns(3,
						consoleRunningCheck,
						consoleStart,
						consoleStop,
						backendRunningCheck,
						backendStart,
						backendStop,
					),
					widget.NewAccordion(widget.NewAccordionItem("Advanced", advancedContainer)),
					createUserText,
					container.NewHBox(
						layout.NewSpacer(),
						createUser,
						layout.NewSpacer(),
					),
					startGameText,
				),
			),
		),
	)
}
