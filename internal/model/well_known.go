package model

type WellKnown struct {
	Version string  `json:"version"`
	RunMode RunMode `json:"runMode"`

	// TODO: Rename the field to ConsoleServerAddr
	Addr            string `json:"consoleServerAddr"`
	RelayServerAddr string `json:"relayServerAddr,omitempty"`

	CallerIP string `json:"callerIP,omitempty"`
}

type RunMode string

const (
	RunModeSinglePlayer RunMode = "single"
	RunModeLAN          RunMode = "lan"
	RunModeRelay        RunMode = "relay-beta"
	RunModeWebRTC       RunMode = "webrtc-beta"
)

func (m RunMode) String() string { return string(m) }
