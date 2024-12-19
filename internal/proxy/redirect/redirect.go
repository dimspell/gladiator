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
	CurrentUserIsHost
	OtherUserIsHost
	OtherUserHasJoined
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
	Run(ctx context.Context, rw io.ReadWriteCloser) error

	io.Writer
	io.Closer
}

type Addressing struct {
	IP      net.IP
	TCPPort string
	UDPPort string
}

type NewRedirect func(joinType Mode, addr *Addressing) (tcpProxy Redirect, udpProxy Redirect, err error)

func New(joinType Mode, addr *Addressing) (tcpProxy Redirect, udpProxy Redirect, err error) {
	slog.Info("Creating redirect", "joinType", joinType.String(), "ip", addr.IP)

	switch joinType {
	case CurrentUserIsHost:
		// All players, who connect to the server are guests (joiners).
		// We are connecting (dialing) to ourselves on the loopback interface,
		// to the local instance served by the DispelMulti.exe.

		slog.Info("Creating client to dial TCP and UDP on default ports", "ip", addr.IP)

		tcpProxy, err = DialTCP(addr.IP.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		udpProxy, err = DialUDP(addr.IP.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		return tcpProxy, udpProxy, nil
	case OtherUserIsHost:
		// The person who is connecting is a host (game creator).
		// We are exposing a packet redirect on the local IP address,
		// to which the game is going to connect (dial).

		slog.Info("Creating TCP and UDP listeners on custom ports", "ip", addr.IP, "tcpPort", addr.TCPPort, "udpPort", addr.UDPPort)
		tcpProxy, err = ListenTCP(addr.IP.To4().String(), addr.TCPPort)
		if err != nil {
			return nil, nil, err
		}
		udpProxy, err = ListenUDP(addr.IP.To4().String(), addr.UDPPort)
		if err != nil {
			return nil, nil, err
		}
		return tcpProxy, udpProxy, nil
	case OtherUserHasJoined:
		// The person who is connecting is a guest, who has already joined.
		// We are connecting (dialing) to the host (game creator) on the loopback interface,
		// to the local instance served by the DispelMulti.exe.

		slog.Info("Creating UDP listener only on a custom port", "ip", addr.IP, "udpPort", addr.UDPPort)
		udpProxy, err = ListenUDP(addr.IP.To4().String(), addr.UDPPort)
		if err != nil {
			return nil, nil, err
		}
		return nil, udpProxy, nil
	case OtherUserIsJoining:
		// The person who is connecting is a guest, who has not joined yet.
		// We have registered the join during the game phase.
		// We are dialing to ourselves on the loopback interface,

		slog.Info("Creating UDP dialler on the default port", "ip", addr.IP)
		udpProxy, err = DialUDP(addr.IP.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		return nil, udpProxy, nil
	default:
		return nil, nil, fmt.Errorf("unknown joining type: %s", joinType)
	}
}

func NewNoop(_ Mode, _ *Addressing) (Redirect, Redirect, error) {
	return &Noop{}, &Noop{}, nil
}
