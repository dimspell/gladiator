package model

import (
	"fmt"
	"net"
)

type WellKnown struct {
	Version string  `json:"version"`
	RunMode RunMode `json:"runMode"`

	// TODO: Rename the field to ConsoleServerAddr
	Addr            string `json:"consoleServerAddr"`
	RelayServerAddr string `json:"relayServerAddr,omitempty"`

	CallerAddr WellKnownCaller `json:"callerAddr,omitempty"`
}

type WellKnownCaller string

func (w WellKnownCaller) IP() (net.IP, error) {
	host, _, err := net.SplitHostPort(string(w))
	if err != nil {
		return nil, fmt.Errorf("could not extract IPv4 from %q: %w", w, err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		return nil, fmt.Errorf("not an IPv4 address: %s", host)
	}
	return ip, nil
}

func (w WellKnownCaller) IPString(fallBackIP string) (string, error) {
	ip, err := w.IP()
	if err != nil {
		return fallBackIP, err
	}
	return ip.String(), nil
}

type RunMode string

const (
	RunModeSinglePlayer RunMode = "single"
	RunModeLAN          RunMode = "lan"
	RunModeRelay        RunMode = "relay-beta"
	RunModeWebRTC       RunMode = "webrtc-beta"
)

func (m RunMode) String() string { return string(m) }
