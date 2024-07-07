package model

type WellKnown struct {
	Version  string `json:"version"`
	Protocol string `json:"protocol"`
	Addr     string `json:"addr"`
	RunMode  string `json:"runMode"`

	CallerInfo WellKnownCallerInfo `json:"callerInfo,omitempty"`
}

type WellKnownCallerInfo struct {
	CallerIP string `json:"callerIP"`
}

type RunMode string

func (m RunMode) String() string { return string(m) }

const (
	RunModeSingle RunMode = "SINGLE_PLAYER"
	RunModeLAN    RunMode = "LAN"
	RunModeHosted RunMode = "HOSTED"
)
