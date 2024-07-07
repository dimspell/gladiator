package ui

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dimspell/gladiator/console/auth"
	"github.com/dimspell/gladiator/console/database"
)

func (c *Controller) SignUnScreen(w fyne.Window) fyne.CanvasObject {
	return container.NewPadded(container.NewVBox(
		headerContainer("Sign-up", func() {
			changePage(w, "JoinOptions", c.JoinOptionsScreen(w))
		}),
		widget.NewLabel(""),
		widget.NewLabel("Provide the credentials how do you want to sign-in."),
		c.signUpForm(nil, nil, w),
	))
}

func (c *Controller) signUpForm(onCancel func(), onCreate func(user database.User), w fyne.Window) *widget.Form {
	name := widget.NewEntry()
	name.Validator = usernameValidator
	name.SetPlaceHolder("GumaTurbo2137")

	password := widget.NewPasswordEntry()
	password.Validator = passwordValidator
	password.SetPlaceHolder("Password")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: name, HintText: "Your login name - alphanumeric, no spaces or special chars"},
			{Text: "Password", Widget: password, HintText: "Use something short and memorable"},
		},
		SubmitText: "Create a new account",
		OnCancel:   onCancel,
		OnSubmit: func() {
			if c.Console == nil {
				slog.Error("Console not initialized")
				return
			}

			loadingDialog := dialog.NewCustomWithoutButtons("Submitting the form...", widget.NewProgressBarInfinite(), w)
			loadingDialog.Show()

			pwd, err := auth.NewPassword(password.Text)
			if err != nil {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}
			user, err := c.Console.DB.Write.CreateUser(context.TODO(), database.CreateUserParams{
				Username: name.Text,
				Password: pwd.String(),
			})
			if err != nil {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}

			loadingDialog.Hide()
			slog.Info("Created new user", "name", user.Username, "id", user.ID)
			if onCreate != nil {
				onCreate(user)
			}
		},
	}

	return form
}
