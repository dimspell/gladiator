//go:build !windows

package ui

func readRegistryKey() (string, error) {
	return "", nil
}

func changeRegistryKey() (likelyChanged bool) {
	return false
}
