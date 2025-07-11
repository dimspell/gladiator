package ui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/console/auth"
)

func (c *Controller) signUpForm(consoleUri string, onCancel func(), onCreate func(user string), w fyne.Window) *widget.Form {
	name := widget.NewEntry()
	name.Validator = usernameValidator
	name.SetPlaceHolder("GumaTurbo2137")

	password := widget.NewPasswordEntry()
	password.Validator = passwordValidator
	password.SetPlaceHolder("Password")

	if !strings.Contains(consoleUri, "//") {
		consoleUri = fmt.Sprintf("%s://%s", "http", consoleUri)
	}
	client := multiv1connect.NewUserServiceClient(&http.Client{Timeout: 5 * time.Second}, consoleUri+"/grpc")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: name, HintText: "Your login name - alphanumeric, no spaces or special chars"},
			{Text: "Password", Widget: password, HintText: "Use something short and memorable"},
		},
		SubmitText: "Create a new account",
		OnCancel:   onCancel,
		OnSubmit: func() {
			loadingDialog := dialog.NewCustomWithoutButtons("Submitting the form...", widget.NewProgressBarInfinite(), w)
			loadingDialog.Show()

			pwd, err := auth.NewPassword(password.Text)
			if err != nil {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			user, err := client.CreateUser(ctx, connect.NewRequest(&v1.CreateUserRequest{
				Username: name.Text,
				Password: pwd.String(),
			}))
			if err != nil {
				loadingDialog.Hide()
				dialog.ShowError(err, w)
				return
			}

			loadingDialog.Hide()
			slog.Info("Created new user", "name", user.Msg.GetUser().Username, "id", user.Msg.GetUser().UserId)
			if onCreate != nil {
				onCreate(user.Msg.GetUser().Username)
			}
		},
	}

	return form
}
