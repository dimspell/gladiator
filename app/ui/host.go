package ui

import (
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HostDatabaseType string

const (
	HostDatabaseTypeSqlite HostDatabaseType = "1_sqlite"
	HostDatabaseTypeMemory HostDatabaseType = "2_memory"
)

func (c *Controller) HostScreen(w fyne.Window) fyne.CanvasObject {
	pathLabel := widget.NewLabelWithStyle("Database Path", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	pathEntry := widget.NewEntry()

	pathSelection := widget.NewButtonWithIcon("Select folder", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if list == nil {
				log.Println("Cancelled")
				return
			}

			pathEntry.SetText(list.Path())
		}, w)
	})

	pathEntry.SetMinRowsVisible(1)
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
			pathEntry.Hide()
			pathSelection.Hide()
		} else {
			pathLabel.Show()
			pathEntry.Show()
			pathSelection.Show()
		}
	})

	typeLabel := widget.NewLabelWithStyle("Database Type", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	typeEntry := comboGroup

	comboGroup.SetSelected(comboOptions[HostDatabaseTypeSqlite])
	pathLabel.Hidden = true
	pathEntry.Hidden = true
	pathSelection.Hidden = true

	pathEntry.Text, _ = defaultDirectory()

	headerText := "Host a server"

	bindLabel := widget.NewLabelWithStyle("Bind Address (IP, Host)", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	bindIP := widget.NewEntry()

	bindIP.Validator = ipValidator
	bindIP.PlaceHolder = "Example: 0.0.0.0"
	bindIP.SetText("127.0.0.1")

	bindPort := widget.NewEntry()

	bindPort.Validator = portValidator
	bindPort.PlaceHolder = "Example: 2137"
	bindPort.SetText("2137")

	bindGroup := container.NewGridWithColumns(2, bindIP, bindPort)

	formGrid := container.New(
		layout.NewFormLayout(),
		bindLabel, bindGroup,
		typeLabel, typeEntry,
		pathLabel, pathContainer,
	)

	onHost := func() {
		if err := bindIP.Validate(); err != nil {
			dialog.NewError(err, w)
			return
		}
		if err := bindPort.Validate(); err != nil {
			dialog.NewError(err, w)
			return
		}

		loadingDialog := dialog.NewCustomWithoutButtons("Starting auth server...", widget.NewProgressBarInfinite(), w)
		loadingDialog.Show()

		databaseType, ok := databaseTypes[comboGroup.Selected]
		if !ok {
			dialog.ShowError(fmt.Errorf("unknown database type: %q", databaseType), w)
			loadingDialog.Hide()
			return
		}
		databasePath := pathEntry.Text + string(os.PathSeparator) + "dispel-multi.sqlite"

		if err := c.StartConsole(databaseType, databasePath, bindIP.Text, bindPort.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}

		loadingDialog.Hide()
		loadingDialog = nil

		// time.AfterFunc(5*time.Second, func() {
		// 	log.Println(syscall.Kill(syscall.Getpid(), syscall.SIGINT))
		// })

		w.SetContent(c.AdminScreen(w))
	}

	btn := widget.NewButtonWithIcon("Submit", theme.NavigateNextIcon(), onHost)
	label := widget.NewRichTextFromMarkdown("**Auth server configuration**\n\nLet's get your game server up and running. Please fill out the following form to specify the configuration details.")
	label.Wrapping = fyne.TextWrapWord
	size := label.Size()
	size.Height /= 4
	label.Resize(size)
	label.Refresh()

	return container.NewStack(
		container.NewBorder(
			container.NewPadded(
				headerContainer(headerText, func() {
					log.Println("Start")
					w.SetContent(c.StartScreen(w, startOptionHost))
				})),
			nil,
			nil,
			nil,
			container.NewPadded(
				container.NewVBox(
					widget.NewLabel(""),
					label,

					container.NewPadded(formGrid),
					container.NewCenter(btn),
				),
			),
		),
	)
}
