package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

var ErrClosed = errors.New("server closed")

type Mode int

const (
	None Mode = iota

	// All players, who connect to the server are guests (joiners).
	// We are connecting (dialing) to ourselves on the loopback interface,
	// to the local instance served by the DispelMulti.exe.
	CurrentUserIsHost

	// The person who is connecting is a host (game creator).
	// We are exposing a packet redirect on the local IP address,
	// to which the game is going to connect (dial).
	OtherUserIsHost

	// The person who is connecting is a guest, who has already joined.
	// We are connecting (dialing) to the host (game creator) on the loopback interface,
	// to the local instance served by the DispelMulti.exe.
	OtherUserHasJoined

	// The person who is connecting is a guest, who has not joined yet.
	// We have registered the join during the game phase.
	// We are dialing to ourselves on the loopback interface,
	OtherUserIsJoining
)

func (s Mode) String() string {
	switch s {
	case None:
		return "None"
	case CurrentUserIsHost:
		return "CurrentUserIsHost"
	case OtherUserIsHost:
		return "OtherUserIsHost"
	case OtherUserHasJoined:
		return "OtherUserHasJoined"
	case OtherUserIsJoining:
		return "OtherUserIsJoining"
	default:
		return "Unknown"
	}
}

type Redirect interface {
	Run(ctx context.Context) error
	Alive(now time.Time, timeout time.Duration) bool

	io.Writer
	io.Closer
}

const defaultUDPPort = "6113"
const defaultTCPPort = "6114"

type Addressing struct {
	IP      net.IP
	TCPPort string
	UDPPort string
}

type NewRedirect func(joinType Mode, addr *Addressing) (Redirect, error)

func NewUDPRedirect(joinType Mode, addr *Addressing) (Redirect, error) {
	logger := slog.With(
		slog.String("redirect", "NewUDPRedirect"),
		slog.String("joinType", joinType.String()),
		slog.String("ip", addr.IP.String()),
		slog.String("udpPort", addr.UDPPort))
	logger.Debug("Creating new UDP redirect")

	switch joinType {
	case CurrentUserIsHost:
		logger.Info("Creating client to dial TCP and UDP on default ports")
		return NewDialUDP(addr.IP.To4().String(), "", nil)
	case OtherUserIsHost:
		logger.Info("Creating TCP and UDP listeners on custom ports")
		return NewListenerUDP(addr.IP.To4().String(), addr.UDPPort, nil)
	case OtherUserHasJoined:
		logger.Info("Creating UDP listener only on a custom port")
		return NewListenerUDP(addr.IP.To4().String(), addr.UDPPort, nil)
	case OtherUserIsJoining:
		logger.Info("Creating UDP dialler on the default port")
		return NewDialUDP(addr.IP.To4().String(), "", nil)
	default:
		return nil, fmt.Errorf("unknown joining type: %s", joinType)
	}
}

func NewTCPRedirect(joinType Mode, addr *Addressing) (Redirect, error) {
	logger := slog.With(
		slog.String("redirect", "NewTCPRedirect"),
		slog.String("joinType", joinType.String()),
		slog.String("ip", addr.IP.String()),
		slog.String("tcpPort", addr.TCPPort))
	logger.Debug("Creating new UDP redirect")

	switch joinType {
	case CurrentUserIsHost:
		logger.Info("Creating client to dial TCP and UDP on default ports")
		return NewDialTCP(addr.IP.To4().String(), "", nil)
	case OtherUserIsHost:
		logger.Info("Creating TCP and UDP listeners on custom ports")
		return NewListenerTCP(addr.IP.To4().String(), addr.TCPPort, nil)
	case OtherUserHasJoined:
		logger.Info("Creating UDP listener only on a custom port")
		return NewListenerUDP(addr.IP.To4().String(), addr.UDPPort, nil)
	default:
		return &Noop{}, nil
	}
}
