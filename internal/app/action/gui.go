//go:build gui

package action

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/dimspell/gladiator/internal/app/ui"
	"github.com/urfave/cli/v3"
)

func GUICommand(version string) *cli.Command {
	cmd := &cli.Command{
		Name:        "gui",
		Description: "Start the GUI app",
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		a := app.NewWithID("com.github.dimspell.gladiator")
		w := a.NewWindow("Gladiator")

		// selfManage(a, w)

		ctrl := ui.NewController(a, version)
		w.SetContent(ctrl.WelcomeScreen(w))

		w.Resize(fyne.NewSize(600, 500))
		w.ShowAndRun()
		return nil
	}
	return cmd
}
