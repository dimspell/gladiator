package main

import (
	"time"

	"github.com/dispel-re/dispel-multi/app"
)

// Version stores what is a current version and git revision of the build.
// See more by using `go version -m ./path/to/binary` command.
var (
	version = "(devel)"
	commit  = ""
	date    = time.Now().UTC().Format(time.RFC3339)
)

func main() {
	app.NewApp(version, commit, date)
}
