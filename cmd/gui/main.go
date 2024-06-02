package main

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/dispel-re/dispel-multi/cmd/gui/internal/ui"
	"github.com/fynelabs/fyneselfupdate"
	"github.com/fynelabs/selfupdate"
	"golang.org/x/crypto/ed25519"
)

func selfManage(a fyne.App, w fyne.Window, sourceURL string) {
	// Used `selfupdatectl create-keys` followed by `selfupdatectl print-key`
	publicKey := ed25519.PublicKey{154, 136, 116, 223, 168, 77, 245, 149, 98, 81, 84, 4, 10, 79, 102, 226, 217, 174, 215, 192, 237, 41, 151, 252, 233, 39, 34, 99, 157, 166, 224, 148}
	httpSource := selfupdate.NewHTTPSource(nil, sourceURL)
	config := fyneselfupdate.NewConfigWithTimeout(a, w, time.Duration(1)*time.Minute,
		httpSource,
		selfupdate.Schedule{FetchOnStart: true, Interval: time.Hour * time.Duration(24)}, // Checking for binary update on start and every 24 hours
		publicKey)
	_, err := selfupdate.Manage(config)
	if err != nil {
		log.Println("Error while setting up update manager: ", err)
		return
	}
}

func main() {
	a := app.NewWithID("net.dispelmulti.app")
	w := a.NewWindow("Dispel Multi")

	ctrl := ui.NewController(a.Storage())
	w.SetContent(ctrl.WelcomeScreen(w))
	// w.SetContent(mainContent.MakeUI())

	w.Resize(fyne.NewSize(600, 500))
	w.ShowAndRun()
}
