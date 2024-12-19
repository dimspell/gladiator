package p2p

import (
	"container/ring"
	"fmt"
	"net"
	"sync"

	"github.com/dimspell/gladiator/internal/proxy/redirect"
)

type IpRing struct {
	Ring *ring.Ring
	mtx  sync.Mutex

	IsTesting bool
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
	if !r.IsTesting {
		return r.NextIP(), "", ""
	}

	ip = net.IPv4(127, 0, 0, 1)

	next := r.NextInt()
	portTCP = fmt.Sprintf("6114%d", next)
	portUDP = fmt.Sprintf("6113%d", next)

	return ip, portTCP, portUDP
}

func (r *IpRing) NextPeerAddress(userId string, isCurrentUser, isHost bool) *Peer {
	switch true {
	case isCurrentUser:
		return &Peer{
			PeerUserID: userId,
			Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:       redirect.CurrentUserIsHost,
		}
	case !isCurrentUser && isHost:
		ip, portTCP, portUDP := r.NextAddr()
		return &Peer{
			PeerUserID: userId,
			Addr:       &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP},
			Mode:       redirect.OtherUserIsHost,
		}
	case !isCurrentUser && !isHost:
		ip, _, portUDP := r.NextAddr()
		return &Peer{
			PeerUserID: userId,
			Addr:       &redirect.Addressing{IP: ip, TCPPort: "", UDPPort: portUDP},
			Mode:       redirect.OtherUserHasJoined,
		}
	default:
		return &Peer{
			PeerUserID: userId,
			Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:       redirect.OtherUserIsJoining,
		}
	}
}
