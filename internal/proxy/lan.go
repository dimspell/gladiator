package proxy

import (
	"fmt"
	"net"
)

var _ Proxy = (*LAN)(nil)

type LAN struct {
	// myIPAddress string
}

func NewLAN() *LAN {
	return &LAN{}
}

func (p *LAN) Create(params CreateParams) (net.IP, error) {
	ip := net.ParseIP(params.HostUserIP)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", params.HostUserIP)
	}
	return ip, nil
}

func (p *LAN) Host(_ HostParams) error {
	return nil
}

func (p *LAN) Join(_ JoinParams) error { return nil }

func (p *LAN) Exchange(params ExchangeParams) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) GetHostIP(hostIpAddress string) net.IP {
	ip := net.ParseIP(hostIpAddress)
	if ip == nil {
		return net.IP{}
	}
	return ip
}

func (p *LAN) Close() {
	// noop
}
