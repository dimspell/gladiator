//go:build !gui

package action

import (
	"github.com/urfave/cli/v3"
)

func GUICommand(_ string) *cli.Command {
	return nil
}
