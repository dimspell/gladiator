package p2p

import (
	"container/ring"
	"fmt"
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

func (r *IpRing) NextAddr() (ip net.IP, portTCP string, portUDP string) {
	if !r.isTesting {
		return r.NextIP(), "", ""
	}

	ip = net.IPv4(127, 0, 0, 1)

	next := r.NextInt()
	portTCP = fmt.Sprintf("6114%d", next)
	portUDP = fmt.Sprintf("6113%d", next)

	return ip, portTCP, portUDP
}

// Deprecated: use redirect.New() instead.
func (r *IpRing) ParseJoiningType(currentUserIsHost bool, other signalserver.Member) (redirect.RedirectType, *redirect.Addressing) {
	switch {
	case currentUserIsHost:
		return redirect.CurrentUserIsHost, &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)}
	case other.IsHost:
		ip, portTCP, portUDP := r.NextAddr()
		return redirect.OtherUserIsHost, &redirect.Addressing{ip, portTCP, portUDP}
	case other.Joined:
		ip, _, portUDP := r.NextAddr()
		return redirect.OtherUserHasJoined, &redirect.Addressing{IP: ip, TCPPort: "", UDPPort: portUDP}
	default:
		return redirect.OtherUserIsJoining, &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)}
	}
}

func (r *IpRing) CreateClient(currentUserIsHost bool, other signalserver.Member) (tcpProxy redirect.Redirect, udpProxy redirect.Redirect, err error) {
	joinType, addr := r.ParseJoiningType(currentUserIsHost, other)
	return redirect.New(joinType, addr)
}
