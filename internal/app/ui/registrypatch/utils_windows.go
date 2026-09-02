//go:build windows

package registrypatch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// The key below holds the server address the installed game connects to.
// PatchRegistry only rewrites it after the user confirms, and
// RestoreRegistry puts the old value back.
const (
	registryPath       = `SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`
	registryKeyServer  = "Server"
	registryKeyVersion = "Version"
)

// backupFile remembers the previous Server value so the change can be undone.
func backupFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "gladiator", "server-backup.txt")
}

func readValue(keyName string) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	s, _, err := key.GetStringValue(keyName)
	if err != nil {
		return "", err
	}
	return s, nil
}

func ReadServer() (string, error) {
	s, err := readValue(registryKeyServer)
	if err != nil {
		return "", fmt.Errorf("could not read the server setting (is the game installed?): %w", err)
	}
	return s, nil
}

// PatchRegistry saves the current Server and Version values, then points
// Server at the local server and sets Version to the expected value. Only
// call it after the user confirmed the change in the UI.
func PatchRegistry() bool {
	if server, err := ReadServer(); err == nil {
		version, _ := readValue(registryKeyVersion)
		_ = os.MkdirAll(filepath.Dir(backupFile()), 0o755)
		_ = os.WriteFile(backupFile(), []byte(server+"\n"+version), 0o600)
	}
	changed := patchRegistryKey(registryKeyServer, "localhost")
	_ = patchRegistryKey(registryKeyVersion, "1.30")
	return changed
}

// RestoreRegistry writes the saved Server and Version values back. It
// returns false when there is nothing to restore.
func RestoreRegistry() bool {
	prev, err := os.ReadFile(backupFile())
	if err != nil || len(prev) == 0 {
		return false
	}
	server, version, _ := strings.Cut(string(prev), "\n")
	ok := patchRegistryKey(registryKeyServer, server)
	if version != "" {
		ok = patchRegistryKey(registryKeyVersion, version) && ok
	}
	return ok
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
