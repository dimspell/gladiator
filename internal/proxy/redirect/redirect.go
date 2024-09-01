package redirect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
)

type RedirectType int

const (
	None RedirectType = iota
	CurrentUserIsHost
	OtherUserIsHost
	OtherUserHasJoined
	OtherUserIsJoining
)

func (s RedirectType) String() string {
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

func New(joinType RedirectType, addr *Addressing) (tcpProxy Redirect, udpProxy Redirect, err error) {
	switch joinType {
	case CurrentUserIsHost:
		// All players, who connect to the server are guests (joiners).
		// We are connecting (dialing) to ourselves on the loopback interface,
		// to the local instance served by the DispelMulti.exe.

		slog.Debug("Creating client to dial TCP and UDP on default ports", "ip", addr.IP)

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

		slog.Debug("Creating TCP and UDP listeners on custom ports", "ip", addr.IP, "tcpPort", addr.TCPPort, "udpPort", addr.UDPPort)

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

		slog.Debug("Creating UDP listener only on a custom port", "ip", addr.IP, "udpPort", addr.UDPPort)
		udpProxy, err = ListenUDP(addr.IP.To4().String(), addr.UDPPort)
		if err != nil {
			return nil, nil, err
		}
		return nil, udpProxy, nil
	case OtherUserIsJoining:
		// The person who is connecting is a guest, who has not joined yet.
		// We have registered the join during the game phase.
		// We are dialing to ourselves on the loopback interface,

		slog.Debug("Creating UDP dialler on the default port", "ip", addr.IP)
		udpProxy, err = DialUDP(addr.IP.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		return nil, udpProxy, nil
	default:
		return nil, nil, fmt.Errorf("unknown joining type: %s", joinType)
	}
}
