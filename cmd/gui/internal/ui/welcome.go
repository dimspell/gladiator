package ui

import (
	"log"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}

	return link
}

func WelcomeScreen(w fyne.Window) fyne.CanvasObject {
	const (
		header1 = "## Greetings, brave adventurer!"
		text1   = "Whether you're stepping into the mystical realms of Dman for the first time or returning for another epic journey, we're thrilled to have you here. Prepare yourself for a world of magic, challenges, and camaraderie."
		header2 = "## Ready to Begin Your Journey?"
		text2   = "Follow the wizard to host your very own server or choose an existing server to join forces and forge alliances as you embark on quests together."
	)

	header1Label := widget.NewRichTextFromMarkdown(header1)
	header1Label.Wrapping = fyne.TextWrapWord

	text1Label := widget.NewLabel(text1)
	text1Label.Wrapping = fyne.TextWrapWord

	header2Label := widget.NewRichTextFromMarkdown(header2)
	header2Label.Wrapping = fyne.TextWrapWord

	text2Label := widget.NewLabel(text2)
	text2Label.Wrapping = fyne.TextWrapWord

	div := container.NewVBox(
		widget.NewRichTextFromMarkdown("# Dispel Multi"),

		header1Label,
		text1Label,
		header2Label,
		text2Label,
		widget.NewLabel(""),

		container.New(
			layout.NewHBoxLayout(),
			layout.NewSpacer(),
			widget.NewButtonWithIcon("Join a server", theme.LoginIcon(), func() {
				log.Println("Join")
				w.SetContent(JoinScreen(w))
			}),
			widget.NewButtonWithIcon("Host a server", theme.ContentAddIcon(), func() {
				log.Println("Host")
				w.SetContent(HostScreen(w))
			}),
			layout.NewSpacer(),
		),

		layout.NewSpacer(),
		container.New(layout.NewHBoxLayout(),
			layout.NewSpacer(),
			widget.NewHyperlink("GitHub", parseURL("https://github.com/dispel-re/dispel-multi")),
			widget.NewLabel("-"),
			widget.NewHyperlink("Discord", parseURL("https://discord.gg/XCNrwvdV6R")),
			widget.NewLabel("-"),
			widget.NewHyperlink("Reddit", parseURL("https://www.reddit.com/r/DispelRPG")),
			layout.NewSpacer(),
		),
		// widget.NewLabel(""), // balance the header on the tutorial screen we leave blank on this content
	)

	return container.NewPadded(div)
}
