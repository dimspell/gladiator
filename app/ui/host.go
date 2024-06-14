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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type HostDatabaseTypeLabel string

const (
	HostDatabaseTypeSqlite HostDatabaseTypeLabel = "1_sqlite"
	HostDatabaseTypeMemory HostDatabaseTypeLabel = "2_memory"
)

var (
	databaseTypeText = map[HostDatabaseTypeLabel]string{
		HostDatabaseTypeSqlite: "Saved on disk (sqlite)",
		HostDatabaseTypeMemory: "Stored in-memory (for testing)",
	}

	databaseTypes = map[string]string{
		databaseTypeText[HostDatabaseTypeSqlite]: "sqlite",
		databaseTypeText[HostDatabaseTypeMemory]: "memory",
	}
)

type HostScreenInputParams struct {
	HostType HostDatabaseTypeLabel
}

func (c *Controller) HostScreen(w fyne.Window, params *HostScreenInputParams) fyne.CanvasObject {
	headerText := "Host a server"

	pathLabel := widget.NewLabelWithStyle("Database Path:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	pathEntry := widget.NewEntry()
	pathEntry.Text, _ = defaultDirectory()
	pathSelection := widget.NewButtonWithIcon("Select Folder", theme.FolderOpenIcon(), selectDatabasePath(w, pathEntry))
	pathContainer := container.NewBorder(nil, nil, nil, pathSelection, pathEntry)

	comboGroup := widget.NewSelect(Values(databaseTypeText), func(value string) {
		log.Println("Select set to", value)

		if value == databaseTypeText[HostDatabaseTypeMemory] {
			pathLabel.Hide()
			pathContainer.Hide()
		} else {
			pathLabel.Show()
			pathContainer.Show()
		}
	})

	switch params.HostType {
	case HostDatabaseTypeSqlite:
		comboGroup.SetSelected(databaseTypeText[HostDatabaseTypeSqlite])
		break
	default:
		comboGroup.SetSelected(databaseTypeText[HostDatabaseTypeMemory])
		pathLabel.Hide()
		pathContainer.Hide()
		break
	}

	bindIP := widget.NewEntry()
	bindIP.Validator = ipValidator
	bindIP.PlaceHolder = "Example: 0.0.0.0"
	bindIP.SetText("0.0.0.0")

	bindPort := widget.NewEntry()
	bindPort.Validator = portValidator
	bindPort.PlaceHolder = "Example: 2137"
	bindPort.SetText("2137")

	formGrid := container.New(
		layout.NewFormLayout(),

		widget.NewLabelWithStyle("Bind Address (IP, Host):", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, bindIP, bindPort),

		widget.NewLabelWithStyle("Database Type:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}),
		comboGroup,

		pathLabel,
		pathContainer,
	)

	onHost := func() {
		if err := validateAll(bindIP.Validate, bindPort.Validate); err != nil {
			dialog.NewError(err, w)
			return
		}

		loadingDialog := dialog.NewCustomWithoutButtons("Starting auth server...", widget.NewProgressBarInfinite(), w)
		loadingDialog.Show()

		databaseType, ok := databaseTypes[comboGroup.Selected]
		if !ok {
			loadingDialog.Hide()
			dialog.ShowError(fmt.Errorf("unknown database type: %q", databaseType), w)
			return
		}

		databasePath := pathEntry.Text
		if err := os.MkdirAll(path.Dir(databasePath), 0755); err != nil {
			if !errors.Is(err, os.ErrExist) {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}
		}

		if err := c.StartConsole(databaseType, databasePath, net.JoinHostPort(bindIP.Text, bindPort.Text)); err != nil {
			loadingDialog.Hide()
			dialog.ShowError(err, w)
			return
		}

		loadingDialog.Hide()
		changePage(w, "Admin", c.AdminScreen(w, &AdminScreenInputParams{
			DatabasePath: databasePath,
			DatabaseType: databaseType,
			BindAddress:  net.JoinHostPort(bindIP.Text, bindPort.Text),
		}))
	}

	btn := widget.NewButtonWithIcon("Start Console", theme.NavigateNextIcon(), onHost)
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
					changePage(w, "Start", c.StartScreen(w, startOptionHost))
				})),
			nil,
			nil,
			nil,
			container.NewPadded(
				container.NewVBox(
					widget.NewLabel(""),
					label,

					container.NewPadded(formGrid),
					widget.NewLabel(""),
					container.NewCenter(btn),
				),
			),
		),
	)
}
