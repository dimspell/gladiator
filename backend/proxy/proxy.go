package proxy

import (
	"net"
	"time"
)

var (
	DefaultConnectionTimeout = 2 * time.Second
)

type Proxy interface {
	// Create is used to start serving the traffic to the game host
	Create(localIPAddress string, hostUser string) (net.IP, error)

	HostGame(gameRoom GameRoom, user User) error

	// Join is used to connect to TCP game host
	Join(gameName string, hostUser string, currentPlayer string, ipAddress string) (net.IP, error)

	// Exchange is used by UDP clients
	Exchange(gameId string, userId string, ipAddress string) (net.IP, error)

	GetHostIP(hostIpAddress string) net.IP
	Close()
}

type GameRoom string

func (r GameRoom) String() string {
	return string(r)
}

type User string

func (u User) String() string {
	return string(u)
}
