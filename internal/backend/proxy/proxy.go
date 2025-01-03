package proxy

import (
	"context"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
)

// Proxy is an interface that defines methods for managing game rooms and player
// connections. It provides functionality for creating and hosting game rooms,
// joining game sessions, and retrieving player IP addresses.
type Proxy interface {
	// GetHostIP is used when game attempts to list the IP address of the game
	// room. This function can be used to override the IP address.
	GetHostIP(net.IP, *bsession.Session) net.IP

	// CreateRoom creates a new game room with the provided parameters and returns
	// the IP address of the game host.
	CreateRoom(CreateParams, *bsession.Session) (net.IP, error)

	// HostRoom creates a new game room with the provided parameters and returns
	// an error if the operation fails.
	HostRoom(HostParams, *bsession.Session) error

	GetPlayerAddr(GetPlayerAddrParams, *bsession.Session) (net.IP, error)

	// Join is used to connect to TCP game host
	Join(JoinParams, *bsession.Session) (net.IP, error)

	Close(session *bsession.Session)

	ExtendWire(session *bsession.Session) MessageHandler
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

type MessageHandler interface {
	Handle(ctx context.Context, payload []byte) error
}
