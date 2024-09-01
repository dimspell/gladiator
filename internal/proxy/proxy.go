package proxy

import (
	"net"
)

type Proxy interface {
	// Create is used to start serving the traffic to the game host
	Create(CreateParams) (net.IP, error)

	Host(HostParams) error

	// Join is used to connect to TCP game host
	Join(JoinParams) error

	// Exchange is used by UDP clients
	Exchange(ExchangeParams) (net.IP, error)

	GetHostIP(hostIpAddress string) net.IP

	Close()
}

type CreateParams struct {
	HostUserIP string
	HostUserID string
	GameID     string
}

type HostParams struct {
	GameID     string
	HostUserID string
}

type JoinParams struct {
	HostUserID    string
	GameID        string
	HostUserIP    string
	CurrentUserID string
}

type ExchangeParams struct {
	GameID    string
	UserID    string
	IPAddress string
}
