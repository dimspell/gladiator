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
	"github.com/dispel-re/dispel-multi/console"
	"github.com/dispel-re/dispel-multi/console/database"
)

func (c *Controller) HostScreen(w fyne.Window) fyne.CanvasObject {
	pathLabel := widget.NewLabel("Database Path:")

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

	pathContainer := container.NewVBox(pathEntry, layout.NewSpacer(), pathSelection)

	comboOptions := []string{
		"Saved on disk (sqlite)",
		"Stored in-memory (for testing)",
	}
	comboGroup := widget.NewSelect(comboOptions, func(value string) {
		log.Println("Select set to", value)

		pathNotUsed := value == comboOptions[1]
		pathLabel.Hidden = pathNotUsed
		pathEntry.Hidden = pathNotUsed
		pathSelection.Hidden = pathNotUsed
	})

	bindLabel := widget.NewLabel("Bind Address:")
	bindEntry := widget.NewEntry()
	bindEntry.PlaceHolder = "Example: 0.0.0.0:2137"
	bindEntry.Text = "127.0.0.1:2137"

	typeLabel := widget.NewLabel("Database Type:")
	typeEntry := comboGroup

	comboGroup.SetSelected(comboOptions[1])
	pathLabel.Hidden = true
	pathEntry.Hidden = true
	pathSelection.Hidden = true

	formGrid := container.New(layout.NewFormLayout(), bindLabel, bindEntry, typeLabel, typeEntry, pathLabel, pathContainer)

	headerText := "Host a server"
	header := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	return container.NewPadded(container.NewVBox(
		container.New(
			layout.NewHBoxLayout(),
			widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
				log.Println("Start")
				w.SetContent(c.StartScreen(w))
			}),
			header,
			layout.NewSpacer(),
		),
		widget.NewLabel(""),

		formGrid,

		widget.NewLabel(""),
		container.NewCenter(
			widget.NewButtonWithIcon("Submit", theme.NavigateNextIcon(), func() {
				loadingContainer := container.NewCenter(
					widget.NewProgressBarInfinite(),
				)
				loadingDialog := dialog.NewCustom("Starting auth server...",
					"Cancel",
					loadingContainer,
					w,
				)
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

				c.Console = console.NewConsole(queries, nil)
				// go func() {
				// 	if err := c.Console.Serve(context.TODO(), bindEntry.Text, ""); err != nil {
				// 		dialog.ShowError(err, w)
				// 		return
				// 	}
				// }()

				loadingDialog.Hide()
				loadingDialog = nil

				// time.AfterFunc(5*time.Second, func() {
				// 	log.Println(syscall.Kill(syscall.Getpid(), syscall.SIGINT))
				// })

				w.SetContent(c.AdminScreen(w))
			}),
		),
	))
}
