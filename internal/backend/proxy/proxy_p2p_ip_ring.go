package proxy

import (
	"container/ring"
	"fmt"
	"net"
	"sync"
)

type IpRing struct {
	Ring *ring.Ring
	mtx  sync.Mutex

	TcpPortPrefix int
	UdpPortPrefix int
	IsTesting     bool
}

func NewIpRing() *IpRing {
	r := ring.New(3)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return &IpRing{
		Ring:          r,
		TcpPortPrefix: 6114,
		UdpPortPrefix: 6113,
	}
}

func (r *IpRing) Reset() {
	// Noop
}

func (r *IpRing) NextInt() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	d := r.Ring.Value.(int)
	r.Ring = r.Ring.Next()
	return d
}

func (r *IpRing) NextAddr() (ip net.IP, portTCP string, portUDP string) {
	if !r.IsTesting {
		ip = net.IPv4(127, 0, 1, byte(r.NextInt()))
		return ip, "", ""
	}

	ip = net.IPv4(127, 0, 0, 1)

	next := r.NextInt()
	portTCP = fmt.Sprintf("%d%d", r.TcpPortPrefix, next)
	portUDP = fmt.Sprintf("%d%d", r.UdpPortPrefix, next)

	return ip, portTCP, portUDP
}
