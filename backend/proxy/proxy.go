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

	// Join is used to connect to TCP game host
	Join(gameId string, currentPlayer string, ipAddress string) (net.IP, error)

	// Exchange is used by UDP clients
	Exchange(gameId string, userId string, ipAddress string) (net.IP, error)

	GetHostIP(hostIpAddress string) net.IP
	Close()
}
