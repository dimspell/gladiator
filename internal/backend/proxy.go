package backend

import (
	"net"
)

type Proxy interface {
	// GetHostIP is used when game attempts to list the IP address of the game
	// room. This function can be used to override the IP address.
	GetHostIP(hostIpAddress string) net.IP

	Create(CreateParams) (net.IP, error)
	Host(HostParams) error

	GetPlayerAddr(GetPlayerAddrParams) (net.IP, error)

	// Join is used to connect to TCP game host
	Join(JoinParams) error

	Close()
}

type CreateParams struct {
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

type GetPlayerAddrParams struct {
	GameID        string
	UserID        string
	IPAddress     string
	CurrentUserID string
	HostUserID    string
}
