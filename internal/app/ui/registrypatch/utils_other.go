//go:build !windows

package registrypatch

func ReadServer() (string, error) {
	return "", nil
}

func PatchRegistry() (likelyChanged bool) {
	return false
}
