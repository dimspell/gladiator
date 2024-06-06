package ui

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
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

	comboOptions := []string{
		"Saved on disk (sqlite)",
		"Stored in-memory (for testing)",
	}
	comboGroup := widget.NewSelect(comboOptions, func(value string) {
		log.Println("Select set to", value)

		if value == comboOptions[1] {
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

	comboGroup.SetSelected(comboOptions[1])
	pathLabel.Hidden = true
	pathEntry.Hidden = true
	pathSelection.Hidden = true

	pathEntry.Text, _ = os.UserHomeDir()

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

		// Configure the database connection
		var (
			db  *database.SQLite
			err error
		)
		switch comboGroup.Selected {
		case comboOptions[0]:
			// sqlite
			db, err = database.NewLocal(
				pathEntry.Text +
					string(os.PathSeparator) +
					"dispel-multi.sqlite")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
		case comboOptions[1]:
			// memory
			db, err = database.NewMemory()
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
		default:
			dialog.ShowError(fmt.Errorf("unknown database type"), w)
			return
		}

		queries, err := db.Queries()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		// Update the database to the latest migration
		if err := database.Seed(queries); err != nil {
			dialog.ShowError(err, w)
			return
		}

		c.Console = console.NewConsole(queries, net.JoinHostPort(bindIP.Text, bindPort.Text))

		go func() {
			if err := c.Console.Serve(context.TODO()); err != nil {
				dialog.ShowError(err, w)
				w.SetContent(c.HostScreen(w))
				return
			}
		}()

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
					w.SetContent(c.StartScreen(w))
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