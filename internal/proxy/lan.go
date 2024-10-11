package proxy

import (
	"fmt"
	"net"
)

var _ Proxy = (*LAN)(nil)

type LAN struct {
	MyIPAddress string
}

func NewLAN(myIPAddress string) *LAN {
	if myIPAddress == "" {
		myIPAddress = "127.0.0.1"
	}

	return &LAN{myIPAddress}
}

func (p *LAN) GetHostIP(hostIpAddress string) net.IP {
	ip := net.ParseIP(hostIpAddress)
	if ip == nil {
		return net.IP{}
	}
	return ip
}

func (p *LAN) Create(_ CreateParams) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}
	return ip, nil
}

func (p *LAN) Host(_ HostParams) error {
	return nil
}

func (p *LAN) Join(_ JoinParams) error { return nil }

func (p *LAN) GetPlayerAddr(params GetPlayerAddrParams) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) Close() {
	// noop
}
