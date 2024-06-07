package ui

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) SinglePlayerScreen(w fyne.Window) fyne.CanvasObject {
	const headerText = "Single Player"
	var (
		consoleAddrIP, consoleAddrPort = "127.0.0.1", "2137"
	)

	pathLabel := widget.NewLabelWithStyle("Database Path", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	pathEntry := widget.NewEntry()
	{
		dir, _ := defaultDirectory()
		pathEntry.SetText(dir)
	}

	pathSelection := widget.NewButtonWithIcon("Select folder", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if list == nil {
				return
			}

			pathEntry.SetText(list.Path() + string(os.PathSeparator) + "dispel-multi.sqlite")
		}, w)
	})

	pathContainer := container.NewBorder(nil, nil, nil, pathSelection, pathEntry)

	comboOptions := map[hostDatabaseType]string{
		hostDatabaseTypeSqlite: "Saved on disk (sqlite)",
		hostDatabaseTypeMemory: "Stored in-memory (for testing)",
	}
	databaseTypes := map[string]string{
		comboOptions[hostDatabaseTypeSqlite]: "sqlite",
		comboOptions[hostDatabaseTypeMemory]: "memory",
	}
	comboGroup := widget.NewSelect(Values(comboOptions), func(value string) {
		log.Println("Select set to", value)

		if value == comboOptions[hostDatabaseTypeMemory] {
			pathLabel.Hide()
			pathContainer.Hide()
		} else {
			pathLabel.Show()
			pathContainer.Show()
		}
	})
	comboGroup.SetSelected(comboOptions[hostDatabaseTypeSqlite])

	advancedContainer := container.NewVBox(
		container.New(
			layout.NewFormLayout(),
			widget.NewLabelWithStyle("Database Type:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
			comboGroup,
			pathLabel,
			pathContainer,
		),
	)

	consoleRunning := binding.NewBool()
	consoleRunningCheck := widget.NewCheckWithData("Console running?", consoleRunning)

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
		if err := c.StartConsole(databaseType, dispelDir, consoleAddrIP, consoleAddrPort); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	consoleStop := widget.NewButtonWithIcon("Stop console", theme.MediaStopIcon(), func() {
		c.StopBackend()
	})

	backendRunning := binding.NewBool()
	backendRunningCheck := widget.NewCheckWithData("Backend running?", backendRunning)
	backendStart := widget.NewButtonWithIcon("Start backend", theme.MediaPlayIcon(), func() {
		if err := c.StartBackend(net.JoinHostPort(consoleAddrIP, consoleAddrPort)); err != nil {
			dialog.ShowError(err, w)
			return
		}
	})
	backendStop := widget.NewButtonWithIcon("Stop backend", theme.MediaStopIcon(), func() {
		c.StopBackend()
	})

	createUser := widget.NewButtonWithIcon("Create New User", theme.AccountIcon(), func() {
	})

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
				log.Println("Welcome")
				w.SetContent(c.StartScreen(w, startOptionPlay))
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