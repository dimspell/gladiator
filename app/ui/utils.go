package ui

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
