package p2p

import (
	"container/ring"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/dimspell/gladiator/internal/backend/proxy/redirect"
)

type IpRing struct {
	Ring *ring.Ring
	mtx  sync.Mutex

	isTesting bool
}

func NewIpRing() *IpRing {
	r := ring.New(3)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return &IpRing{Ring: r}
}

func (r *IpRing) NextInt() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	d := r.Ring.Value.(int)
	r.Ring = r.Ring.Next()
	return d
}

func (r *IpRing) NextIP() net.IP {
	return net.IPv4(127, 0, 1, byte(r.NextInt()))
}

func (r *IpRing) NextAddr() (ip string, portTCP string, portUDP string) {
	if !r.isTesting {
		return r.NextIP().To4().String(), "", ""
	}

	ip = net.IPv4(127, 0, 0, 1).To4().String()

	next := r.NextInt()
	portTCP = fmt.Sprintf("6114%d", next)
	portUDP = fmt.Sprintf("6113%d", next)

	return ip, portTCP, portUDP
}

func (r *IpRing) CreateClient(currentUserIsHost bool, other signalserver.Member) (tcpProxy redirect.Redirect, udpProxy redirect.Redirect, err error) {
	if currentUserIsHost {
		// All players, who connect to the server are guests (joiners).
		// We are connecting (dialing) to ourselves on the loopback interface,
		// to the local instance served by the DispelMulti.exe.

		ip := net.IPv4(127, 0, 0, 1)
		slog.Debug("Creating client to dial TCP and UDP on default ports", "ip", ip)

		tcpProxy, err = redirect.DialTCP(ip.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		udpProxy, err = redirect.DialUDP(ip.To4().String(), "")
		if err != nil {
			return nil, nil, err
		}
		return tcpProxy, udpProxy, nil
	}

	if other.IsHost {
		// The person who is connecting is a host (game creator).
		// We are exposing a packet redirect on the local IP address,
		// to which the game is going to connect (dial).

		ip, portTCP, portUDP := r.NextAddr()
		slog.Debug("Creating TCP and UDP listeners on custom ports", "ip", ip, "tcpPort", portTCP, "udpPort", portUDP)

		tcpProxy, err = redirect.ListenTCP(ip, portTCP)
		if err != nil {
			return nil, nil, err
		}
		udpProxy, err = redirect.ListenUDP(ip, portUDP)
		if err != nil {
			return nil, nil, err
		}
		return tcpProxy, udpProxy, nil
	}

	if other.Joined {
		// The person who is connecting is a guest, who has already joined.
		// We are connecting (dialing) to the host (game creator) on the loopback interface,
		// to the local instance served by the DispelMulti.exe.

		ip, _, portUDP := r.NextAddr()
		slog.Debug("Creating UDP listener only on a custom port", "ip", ip, "udpPort", portUDP)

		udpProxy, err = redirect.ListenUDP(ip, portUDP)
		if err != nil {
			return nil, nil, err
		}
		return nil, udpProxy, nil
	}

	// The person who is connecting is a guest, who has not joined yet.
	// We have registered the join during the game phase.
	// In the rest of the cases, we are dialing to ourselves on the loopback
	// interface,

	ip := net.IPv4(127, 0, 0, 1)
	slog.Debug("Creating UDP dialler on the default port", "ip", ip)

	udpProxy, err = redirect.DialUDP(ip.To4().String(), "")
	if err != nil {
		return nil, nil, err
	}
	return nil, udpProxy, nil
}
