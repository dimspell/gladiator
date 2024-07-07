package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows/registry"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
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
	// directoryPath += string(os.PathSeparator)
	directoryPath += "/"
	directoryPath += "dispel-multi"
	// directoryPath += string(os.PathSeparator)
	directoryPath += "/"
	// directoryPath += "dispel-multi.sqlite"
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

func listAllIPs() ([]net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("could not list all interfaces (are there any?): %w", err)
	}

	var ips []net.IP
	for _, iface := range interfaces {
		address, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range address {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.IsLoopback() || !ipNet.IP.IsGlobalUnicast() {
				continue
			}

			ips = append(ips, ipNet.IP)
		}
	}

	return ips, nil
}

const (
	registryPath = `SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`
	registryKey  = "Server"
)

func readRegistryKey() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("could not find the registry key (is the game installed?): %w", err)
	}
	defer key.Close()

	s, _, err := key.GetStringValue(registryKey)
	if err != nil {
		return "", fmt.Errorf("could not read from the %q registry key: %w", registryKey, err)
	}
	return s, nil
}

func changeRegistryKey() (likelyChanged bool) {
	cmd := "reg.exe"
	newValue := "localhost"
	args := strings.Join([]string{
		"ADD",
		fmt.Sprintf(`HKEY_LOCAL_MACHINE\%s`, registryPath),
		"/v", "Server",
		"/t", "REG_SZ",
		"/f",
		"/d", newValue,
	}, " ")

	r := exec.Command("powershell.exe", "Start-Process", cmd, "-Verb", "runAs", "-ArgumentList", `"`+args+`"`)

	// TODO: On any failure (like cancel on UAC propmpt), the powershell will print out the reason to the STDERR. Log it.
	//r.Stdout = os.Stdout
	//r.Stderr = os.Stderr

	// When the user has cancelled the UAC prompt, then the process will have the exit-code=1, so also non-nil error.
	// On this step, it is not possible to detect if the reg.exe has successfully replaced the registry key with new value.
	likelyChanged = r.Run() == nil
	return
}
