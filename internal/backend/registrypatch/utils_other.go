//go:build !windows

package registrypatch

func ReadRegistryKey() (string, error) {
	return "", nil
}

func PatchRegistry() (likelyChanged bool) {
	return false
}
