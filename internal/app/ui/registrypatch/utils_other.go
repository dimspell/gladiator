//go:build !windows

package registrypatch

// The registry helper is Windows-only; the functions below are stubs so
// non-Windows builds compile.

func ReadServer() (string, error) {
	return "", nil
}

func PatchRegistry() (likelyChanged bool) {
	return false
}

func RestoreRegistry() bool {
	return false
}
