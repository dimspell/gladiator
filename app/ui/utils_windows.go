//go:build windows

package ui

import (
	"fmt"
	"golang.org/x/sys/windows/registry"
	"os/exec"
	"strings"
)

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
