//go:build !gui

package action

import (
	"github.com/urfave/cli/v3"
)

func GUICommand() *cli.Command {
	return nil
}
