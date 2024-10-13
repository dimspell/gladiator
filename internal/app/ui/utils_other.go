//go:build !windows

package ui

func readRegistryKey() (string, error) {
	return "", nil
}

func patchRegistryKey() (likelyChanged bool) {
	return false
}
