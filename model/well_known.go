package model

type WellKnown struct {
	Addr    string `json:"addr"`
	RunMode string `json:"runMode"`
}

const (
	RunModeSingle = "SINGLE_PLAYER"
	RunModeLAN    = "LAN"
	RunModeHosted = "HOSTED"
)
