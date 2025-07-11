package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) JoinScreen(w fyne.Window) fyne.CanvasObject {
	headerText := "Join a server"

	consoleAddr := widget.NewEntry()
	consoleAddr.PlaceHolder = "Example: https://multi.example.com"
	consoleAddr.SetText("http://127.0.0.1:2137")

	formGrid := container.New(
		layout.NewFormLayout(),
		widget.NewLabelWithStyle("Server Address", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true}), consoleAddr,
	)

	label := widget.NewRichTextFromMarkdown("**Specify the address**\n\nSpecify the address of the authentication server you wish to connect to. To troubleshoot, you could ask your friend to give you all the IP addresses that can be found after running the _ipconfig_ command on his/her machine.")
	label.Wrapping = fyne.TextWrapWord
	size := label.Size()
	size.Height /= 4
	label.Resize(size)
	label.Refresh()

	return container.NewBorder(
		container.NewPadded(headerContainer(headerText, func() {
			changePage(w, "Start", c.StartScreen(w, startOptionJoin))
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
						loadingDialog := dialog.NewCustomWithoutButtons("Connecting to auth server...", widget.NewProgressBarInfinite(), w)
						loadingDialog.Show()

						metadata, err := c.ConsoleHandshake(consoleAddr.Text)
						if err != nil {
							loadingDialog.Hide()
							dialog.ShowError(err, w)
							return
						}
						loadingDialog.Hide()

						changePage(w, "Joined", c.PlayScreen(w, consoleAddr.Text, metadata))
					}),
				),
			),
		),
	)
}
