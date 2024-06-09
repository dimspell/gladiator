package ui

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func Keys[K comparable, V any](m map[K]V) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

func Values[K ~string, V any](m map[K]V) []V {
	r := make([]V, 0, len(m))

	// Sort keys
	keys := Keys(m)
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, key := range keys {
		r = append(r, m[key])
	}
	return r
}

func headerContainer(headerText string, backCallback func()) *fyne.Container {
	return container.New(
		layout.NewHBoxLayout(),
		widget.NewButtonWithIcon("Go back", theme.NavigateBackIcon(), backCallback),
		widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)
}

var usernameValidator = validation.NewRegexp("[a-zA-Z0-9]{4,24}", "must be alphanumeric up to 24 chars - a-Z / 0-9")

func passwordValidator(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

func ipValidator(s string) error {
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Errorf("invalid IP address")
	}
	return nil
}

func portValidator(s string) error {
	i, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	if i < 1000 || i > 65535 {
		return fmt.Errorf("invalid port number")
	}
	return nil
}

func parseURL(urlStr string) *url.URL {
	link, err := url.Parse(urlStr)
	if err != nil {
		fyne.LogError("Could not parse URL", err)
	}

	return link
}

func validateAll(validators ...func() error) error {
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func defaultDirectory() (string, error) {
	directoryPath, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	directoryPath += string(os.PathSeparator)
	directoryPath += "dispel-multi"
	directoryPath += string(os.PathSeparator)
	directoryPath += "dispel-multi.sql"
	return directoryPath, nil
}

func selectDatabasePath(w fyne.Window, pathEntry *widget.Entry) func() {
	return func() {
		dialog.ShowFolderOpen(func(list fyne.ListableURI, err error) {
			if err := insertDatabasePath(list, err, func(filePath string) {
				pathEntry.SetText(filePath)
			}); err != nil {
				dialog.ShowError(err, w)
			}
		}, w)
	}
}

func insertDatabasePath(list fyne.ListableURI, err error, setFn func(string)) error {
	if err != nil {
		return err
	}
	if list == nil {
		return nil
	}

	setFn(list.Path() + string(os.PathSeparator) + "dispel-multi.sqlite")
	return nil
}

func changePage(w fyne.Window, pageName string, content fyne.CanvasObject) {
	slog.Debug("Changing page in the launcher", "page", pageName)
	w.SetContent(content)
}
