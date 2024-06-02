package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

func (c *Controller) SignUnScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewPadded(container.NewVBox(
		headerContainer("Sign-up", func() {
			log.Println("Join")
			w.SetContent(c.JoinOptionsScreen(w))
		}),
		widget.NewLabel(""),
		widget.NewLabel("Provide the credentials how do you want to sign-in."),
		signUpForm(),
	))
}

func signUpForm() *widget.Form {
	name := widget.NewEntry()
	name.Validator = validation.NewRegexp("[a-zA-Z0-9]{4,24}", "must be alphanumeric up to 24 chars - a-Z / 0-9")
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
			{Text: "Name", Widget: name, HintText: "Your login name"},
			{Text: "Password", Widget: password, HintText: "Use something short and memorable"},
		},
		SubmitText: "Create a new account",
		OnSubmit: func() {
			fmt.Println("Form submitted")
		},
	}

	return form
}
