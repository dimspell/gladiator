package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/dispel-re/dispel-multi/cmd/gui/internal/ui"
)

func main() {
	a := app.NewWithID("net.dispelmulti.app")
	w := a.NewWindow("Dispel Multi")
	// mainContent := ui.CreateMainContent(w, a.Storage())

	// w.SetContent(widget.NewLabel("Hello World!"))
	// w.SetContent(ui.AdminScreen(w))
	w.SetContent(ui.JoinScreen(w))

	// w.SetContent(mainContent.MakeUI())
	w.Resize(fyne.NewSize(600, 400))
	// err := mainContent.StagerController.TakeOver(mainContent.HomeView.GetStageName())
	// if err != nil {
	// 	dialog.ShowError(err, w)
	// }
	w.ShowAndRun()

	// myApp := app.New()
	// myWindow := myApp.NewWindow("Box Layout")
	//
	// text1 := canvas.NewText("Hello", color.White)
	// text2 := canvas.NewText("There", color.White)
	// text3 := canvas.NewText("(right)", color.White)
	// content := container.New(layout.NewHBoxLayout(), text1, text2, layout.NewSpacer(), text3)
	//
	// text4 := canvas.NewText("centered", color.White)
	// centered := container.New(layout.NewHBoxLayout(), layout.NewSpacer(), text4, layout.NewSpacer())
	// myWindow.SetContent(container.New(layout.NewVBoxLayout(), content, centered))
	// myWindow.ShowAndRun()
}
