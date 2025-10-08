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
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
	"github.com/dimspell/gladiator/internal/model"
)

func (c *Controller) PlayScreen(w fyne.Window, consoleAddr string, metadata *model.WellKnown) fyne.CanvasObject {
	return container.NewBorder(
		container.NewPadded(
			headerContainer("Join & Play Multiplayer", func() {
				changePage(w, "Join", c.JoinScreen(w, consoleAddr))
			}),
		),
		nil,
		nil,
		nil,
		c.playView(w, consoleAddr, metadata),
	)
}

func (c *Controller) playView(w fyne.Window, consoleAddr string, metadata *model.WellKnown) fyne.CanvasObject {
	ips, _ := listAllIPs()

	myIPEntry := widget.NewSelectEntry(ips)
	myIPEntry.Validator = ipValidator
	myIPEntry.PlaceHolder = "Example: 192.168.100.1"

	var myIpAddress string
	if len(ips) > 0 {
		myIpAddress = ips[0]
	}
	if myIpAddress == "" {
		myIpAddress = metadata.CallerIP
	}
	myIPEntry.SetText(myIpAddress)

	settingsWidgets := []fyne.CanvasObject{}
	if metadata.RunMode == model.RunModeLAN {
		settingsWidgets = append(settingsWidgets,
			widget.NewLabelWithStyle("My in-game IP Address:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: false}),
			myIPEntry,
		)
	}
	settingsWidgets = append(settingsWidgets,
		widget.NewLabelWithStyle("Auth Server (Console) Address:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: false}),
		widget.NewLabel(consoleAddr),
		widget.NewLabelWithStyle("Configuration Run Mode:", fyne.TextAlignTrailing, fyne.TextStyle{Bold: false}),
		widget.NewLabel(metadata.RunMode.String()),
	)

	settingsAccordion := widget.NewAccordionItem("Settings",
		container.New(
			layout.NewFormLayout(),
			settingsWidgets...,
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

		var proxyCreator backend.ProxyFactory
		switch metadata.RunMode {
		case model.RunModeRelay:
			proxyCreator = &relay.ProxyRelay{RelayServerAddr: metadata.RelayServerAddr}
		default:
			proxyCreator = &direct.ProxyLAN{MyIPAddress: myIPEntry.Text}
		}

		if err := c.StartBackend(consoleAddr, proxyCreator); err != nil {
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
