package app

import "runtime/debug"

func vcsRevision(value string, defaultValue string) string {
	if value != "" {
		return value
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return defaultValue
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return defaultValue
}
