package redirect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
)

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
	Run(ctx context.Context, rw io.Writer) error

	io.Writer
	io.Closer
}

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
		return DialUDP(addr.IP.To4().String(), "")
	case OtherUserIsHost:
		logger.Info("Creating TCP and UDP listeners on custom ports")
		return ListenUDP(addr.IP.To4().String(), addr.UDPPort)
	case OtherUserHasJoined:
		logger.Info("Creating UDP listener only on a custom port")
		return ListenUDP(addr.IP.To4().String(), addr.UDPPort)
	case OtherUserIsJoining:
		logger.Info("Creating UDP dialler on the default port")
		return DialUDP(addr.IP.To4().String(), "")
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
		return DialTCP(addr.IP.To4().String(), "")
	case OtherUserIsHost:
		logger.Info("Creating TCP and UDP listeners on custom ports")
		return ListenTCP(addr.IP.To4().String(), addr.TCPPort)
	case OtherUserHasJoined:
		logger.Info("Creating UDP listener only on a custom port")
		return ListenUDP(addr.IP.To4().String(), addr.UDPPort)
	default:
		return &Noop{}, nil
	}
}
