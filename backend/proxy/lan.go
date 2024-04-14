package proxy

import (
	"fmt"
	"net"
)

var _ Proxy = (*LAN)(nil)

type LAN struct {
}

func NewLAN() *LAN {
	return &LAN{}
}

func (p *LAN) Create(localIPAddress string, _ string) (net.IP, error) {
	ip := net.ParseIP(localIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", localIPAddress)
	}
	return ip, nil
}

func (p *LAN) Join(_ string, _ string, _ string, ipAddress string) (net.IP, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", ipAddress)
	}
	return ip, nil
}

func (p *LAN) Exchange(_ string, _ string, ipAddress string) (net.IP, error) {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", ipAddress)
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
