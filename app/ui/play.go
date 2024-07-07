package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) PlayScreen(w fyne.Window, consoleAddr string, myIpAddress string) fyne.CanvasObject {
	return container.NewBorder(
		container.NewPadded(
			headerContainer("Join & Play Multiplayer", func() {
				changePage(w, "JoinOptions", c.JoinOptionsScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		c.playView(w, consoleAddr, myIpAddress),
	)
}

func (c *Controller) playView(w fyne.Window, consoleAddr, myIpAddress string) fyne.CanvasObject {
	myIPEntry := widget.NewEntry()
	myIPEntry.Validator = ipValidator
	myIPEntry.PlaceHolder = "Example: 192.168.100.1"
	myIPEntry.SetText(myIpAddress)

	settingsAccordion := widget.NewAccordionItem("Settings",
		container.New(
			layout.NewFormLayout(),
			widget.NewLabelWithStyle("My in-game IP Address:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: false}),
			myIPEntry,
			widget.NewLabelWithStyle("Auth Server (Console) Address:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: false}),
			widget.NewLabel(consoleAddr),
		))
	settingsAccordion.Open = true

	backendRunningLabel := binding.NewString()
	backendRunningCheck := widget.NewLabelWithData(backendRunningLabel)
	backendRunningCheck.Alignment = fyne.TextAlignCenter
	backendStart := widget.NewButtonWithIcon("Start backend", theme.MediaPlayIcon(), func() {
		if err := validateAll(myIPEntry.Validate); err != nil {
			dialog.ShowError(err, w)
			return
		}

		loadingDialog := dialog.NewCustomWithoutButtons("Starting backend...", widget.NewProgressBarInfinite(), w)
		loadingDialog.Show()

		if err := c.StartBackend(consoleAddr, myIPEntry.Text); err != nil {
			loadingDialog.Hide()
			return
		}
		loadingDialog.Hide()
	})
	backendStop := widget.NewButtonWithIcon("Stop backend", theme.MediaStopIcon(), func() {
		c.StopBackend()
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

	createUserText := widget.NewRichTextFromMarkdown("**(Optional) Create new user**")
	createUserText.Wrapping = fyne.TextWrapWord
	createUser := widget.NewButtonWithIcon("Create New User", theme.AccountIcon(), func() {
		centered := container.NewCenter()
		d := dialog.NewCustomWithoutButtons("Create New User", centered, w)
		centered.Add(c.signUpForm(consoleAddr, d.Hide, func(username string) {
			d.Hide()

			c.app.SendNotification(
				fyne.NewNotification("Created New User",
					fmt.Sprintf("You have successfully created a new user named %q.", username),
				))
		}, w))
		d.Show()
	})

	startBackendCard := container.NewVBox(
		container.NewGridWithColumns(3,
			backendRunningCheck,
			backendStart,
			backendStop,
		),
		widget.NewAccordion(settingsAccordion),
	)

	return container.NewVScroll(container.NewPadded(
		container.NewVBox(
			renderRegistryNotes(),
			renderRegistryPatchContainer(w),
			renderStartBackendServer(),
			startBackendCard,
			renderCreateUserNotes(),
			container.NewHBox(
				layout.NewSpacer(),
				createUser,
				layout.NewSpacer(),
			),
			renderStartGameNotes(),
		),
	))
}

func renderStartBackendServer() *widget.RichText {
	serversRunningText := widget.NewRichTextFromMarkdown("**2. Start backend servers**\n\n" +
		"You must have the backend running on your computer before starting the game. " +
		"Configure the parameters and start it by clicking on the button:")
	serversRunningText.Wrapping = fyne.TextWrapWord

	return serversRunningText
}
