package proxy

import (
	"net"
)

type Proxy interface {
	// Create is used to start serving the traffic to the game host
	Create(CreateParams) (net.IP, error)

	Host(HostParams) error

	// Join is used to connect to TCP game host
	Join(JoinParams) (net.IP, error)

	// Exchange is used by UDP clients
	Exchange(ExchangeParams) (net.IP, error)

	GetHostIP(hostIpAddress string) net.IP

	Close()
}

type CreateParams struct {
	LocalIP  string
	HostUser string
	// GameRoom string
}

type HostParams struct {
	GameRoom string
	User     string
}

type JoinParams struct {
	HostUser    string
	CurrentUser string
	GameName    string
	IPAddress   string
}

type ExchangeParams struct {
	GameId    string
	UserId    string
	IPAddress string
}
