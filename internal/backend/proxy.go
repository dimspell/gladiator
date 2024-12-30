package backend

import (
	"net"
)

type Proxy interface {
	// GetHostIP is used when game attempts to list the IP address of the game
	// room. This function can be used to override the IP address.
	GetHostIP(hostIpAddress string, session *Session) net.IP

	CreateRoom(CreateParams, *Session) (net.IP, error)
	HostRoom(HostParams, *Session) error

	GetPlayerAddr(GetPlayerAddrParams, *Session) (net.IP, error)

	// Join is used to connect to TCP game host
	Join(JoinParams, *Session) (net.IP, error)

	Close(session *Session)

	ExtendWire(session *Session) MessageHandler
}

type CreateParams struct {
	GameID string
}

type HostParams struct {
	GameID string
}

type JoinParams struct {
	HostUserID string
	GameID     string
	HostUserIP string
}

type GetPlayerAddrParams struct {
	GameID     string
	UserID     string
	IPAddress  string
	HostUserID string
}
