package model

type WellKnown struct {
	Addr    string `json:"addr"`
	RunMode string `json:"runMode"`
}

type RunMode string

func (m RunMode) String() string { return string(m) }

const (
	RunModeSingle RunMode = "SINGLE_PLAYER"
	RunModeLAN    RunMode = "LAN"
	RunModeHosted RunMode = "HOSTED"
)
