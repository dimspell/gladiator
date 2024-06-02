package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) JoinOptionsScreen(w fyne.Window) fyne.CanvasObject {
	headerText := "Join a server"

	radioOptions := map[string]string{
		// "dispelmulti": "Use dispelmulti.net network",
		"loopback": "Use loopback for testing (127.0.0.1:2137)",
		"define":   "Use LAN network - provide the address",
	}
	radioGroup := widget.NewRadioGroup(Values(radioOptions), func(value string) {
		log.Println("Radio set to", value)
	})
	radioGroup.SetSelected(radioOptions["loopback"])
	radioGroup.Required = true

	return container.NewBorder(
		container.NewPadded(
			headerContainer(headerText, func() {
				log.Println("Start")
				w.SetContent(c.StartScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewVBox(
				widget.NewLabel(""),

				widget.NewLabelWithStyle("To which server would you like to connect to?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				radioGroup,

				widget.NewLabel(""),
				container.NewCenter(
					widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), func() {
						log.Println(radioGroup.Selected)
						if radioGroup.Selected == radioOptions["loopback"] {
							consoleAddr := "127.0.0.1:2137"

							loadingDialog := dialog.NewCustomWithoutButtons("Connecting to auth server...", widget.NewProgressBarInfinite(), w)
							loadingDialog.Show()

							if err := c.ConsoleHandshake(consoleAddr); err != nil {
								loadingDialog.Hide()
								dialog.ShowError(err, w)
								return
							}

							if err := c.StartBackend(consoleAddr); err != nil {
								loadingDialog.Hide()
								dialog.ShowError(err, w)
								return
							}

							loadingDialog.Hide()
							w.SetContent(c.JoinedScreen(w))
							return
						}
						if radioGroup.Selected == radioOptions["define"] {
							w.SetContent(c.JoinDefineScreen(w))
							return
						}
					}),
				),
			),
		),
	)
}
