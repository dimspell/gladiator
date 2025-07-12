//go:build windows

package registrypatch

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	registryPath       = `SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`
	registryKeyServer  = "Server"
	registryKeyVersion = "Version"
)

func ReadServer() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("could not find the registry key (is the game installed?): %w", err)
	}
	defer key.Close()

	s, _, err := key.GetStringValue(registryKeyServer)
	if err != nil {
		return "", fmt.Errorf("could not read from the %q registry key: %w", registryKeyServer, err)
	}
	return s, nil
}

func PatchRegistry() bool {
	changed := patchRegistryKey(registryKeyServer, "localhost")
	_ = patchRegistryKey(registryKeyVersion, "1.30")
	return changed
}

func patchRegistryKey(registryKey, newValue string) (likelyChanged bool) {
	cmd := "reg.exe"
	args := strings.Join([]string{
		"ADD",
		fmt.Sprintf(`HKEY_LOCAL_MACHINE\%s`, registryPath),
		"/v", registryKey,
		"/t", "REG_SZ",
		"/f",
		"/d", newValue,
	}, " ")

	r := exec.Command("powershell.exe", "Start-Process", cmd, "-Verb", "runAs", "-ArgumentList", `"`+args+`"`)

	// TODO: On any failure (like cancel on UAC propmpt), the powershell will print out the reason to the STDERR. Log it.
	// r.Stdout = os.Stdout
	// r.Stderr = os.Stderr

	// When the user has cancelled the UAC prompt, then the process will have the exit-code=1, so also non-nil error.
	// On this step, it is not possible to detect if the reg.exe has successfully replaced the registry key with new value.
	likelyChanged = r.Run() == nil
	return
}
