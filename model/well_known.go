package model

import (
	"fmt"
	"net"
)

type WellKnown struct {
	Version  string `json:"version"`
	Protocol string `json:"protocol"`
	Addr     string `json:"addr"`
	RunMode  string `json:"runMode"`

	Caller WellKnownCaller `json:"caller,omitempty"`
}

type WellKnownCaller struct {
	Addr string `json:"addr"`
}

func (w *WellKnownCaller) IP() (net.IP, error) {
	host, _, err := net.SplitHostPort(w.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not extract IPv4 from %q: %w", w.Addr, err)
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		return nil, fmt.Errorf("not an IPv4 address: %s", host)
	}
	return ip, nil
}

func (w *WellKnownCaller) IPString(fallBackIP string) (string, error) {
	ip, err := w.IP()
	if err != nil {
		return fallBackIP, err
	}
	return ip.String(), nil

}

type RunMode string

const (
	RunModeSingle RunMode = "SINGLE_PLAYER"
	RunModeLAN    RunMode = "LAN"
	RunModeHosted RunMode = "HOSTED"
)

func (m RunMode) String() string { return string(m) }
