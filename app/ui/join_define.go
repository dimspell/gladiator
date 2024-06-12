package ui

import (
	"net"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) JoinDefineScreen(w fyne.Window) fyne.CanvasObject {
	headerText := "Join a server - connect to custom auth server"

	bindIP := widget.NewEntry()

	bindIP.Validator = ipValidator
	bindIP.PlaceHolder = "Example: 0.0.0.0"
	bindIP.SetText("127.0.0.1")

	bindPort := widget.NewEntry()

	bindPort.Validator = portValidator
	bindPort.PlaceHolder = "Example: 2137"
	bindPort.SetText("2137")

	bindGroup := container.NewGridWithColumns(2, bindIP, bindPort)

	bindLabel := widget.NewLabelWithStyle("Server Address (IP, Port)", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})

	formGrid := container.New(
		layout.NewFormLayout(),
		bindLabel, bindGroup,
	)

	label := widget.NewRichTextFromMarkdown("**Specify the address**\n\nSpecify the address of the authentication server you wish to connect to. To troubleshoot, you could ask your friend to give you all the IP addresses that can be found after running the _ipconfig_ command on his/her machine.")
	label.Wrapping = fyne.TextWrapWord
	size := label.Size()
	size.Height /= 4
	label.Resize(size)
	label.Refresh()

	return container.NewBorder(
		container.NewPadded(headerContainer(headerText, func() {
			changePage(w, "JoinOptions", c.JoinOptionsScreen(w))
		})),
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewVBox(
				widget.NewLabel(""),

				label,
				formGrid,

				widget.NewLabel(""),
				container.NewCenter(
					widget.NewButtonWithIcon("Connect", theme.NavigateNextIcon(), func() {
						if err := bindIP.Validate(); err != nil {
							dialog.NewError(err, w)
							return
						}
						if err := bindPort.Validate(); err != nil {
							dialog.NewError(err, w)
							return
						}
						consoleAddr := net.JoinHostPort(bindIP.Text, bindPort.Text)

						// loadingDialog := dialog.NewCustomWithoutButtons("Connecting to auth server...", widget.NewProgressBarInfinite(), w)
						// loadingDialog.Show()
						// defer loadingDialog.Hide()

						if err := c.ConsoleHandshake(consoleAddr); err != nil {
							dialog.ShowError(err, w)
							return
						}

						changePage(w, "Joined", c.PlayScreen(w, consoleAddr))
					}),
				),
			),
		),
	)
}
