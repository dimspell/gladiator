package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) SignInScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewBorder(
		container.NewPadded(
			headerContainer("Sign-up", func() {
				changePage(w, "Join", c.JoinScreen(w))
			}),
		),
		nil,
		nil,
		nil,
		widget.NewLabel(""),
		widget.NewLabel("Provide login & password to sign in:"),
		signInForm(),
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
