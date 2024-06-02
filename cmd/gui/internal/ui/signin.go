package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) SignInScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewBorder(
		container.NewPadded(
			headerContainer("Sign-up", func() {
				log.Println("Join")
				w.SetContent(c.JoinOptionsScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		widget.NewLabel(""),
		// widget.NewLabel("Provide login & password to sign in:"),
		// signInForm(),

		widget.NewButton("Connect", func() {
			todoConsoleAddr := "127.0.0.1:2137"

			if err := c.ConsoleHandshake(todoConsoleAddr); err != nil {
				log.Println(err)
				return
			}

			if err := c.StartBackend(todoConsoleAddr, ""); err != nil {
				log.Println(err)
				return
			}
		}),
	)
}

func signInForm() *widget.Form {
	name := widget.NewEntry()
	name.Validator = func(s string) error {
		if len(s) < 4 {
			return fmt.Errorf("invalid name")
		}
		return nil
	}
	name.SetPlaceHolder("GumaTurbo2137")

	password := widget.NewPasswordEntry()
	password.Validator = func(s string) error {
		if len(s) == 0 {
			return fmt.Errorf("password cannot be empty")
		}
		return nil
	}
	password.SetPlaceHolder("Password")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: name},
			{Text: "Password", Widget: password},
		},
		SubmitText: "Log-in",
		OnSubmit: func() {
			fmt.Println("Form submitted")
		},
	}

	return form
}
