package ui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func SignInScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewPadded(container.NewVBox(
		widget.NewRichTextFromMarkdown("# Dispel Multi"),
		widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), func() {
			log.Println("Welcome")
			w.SetContent(WelcomeScreen(w))
		}),
	))
}
