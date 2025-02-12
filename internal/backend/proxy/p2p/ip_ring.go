package p2p

import (
	"container/ring"
	"fmt"
	"net"
	"sync"
)

const (
	ringSize      = 3
	ipStart       = 2
	localhost     = "127.0.0.1"
	maxPortNumber = 65535
)

// IpRing manages a circular buffer of IP addresses and ports for P2P connections
type IpRing struct {
	Ring *ring.Ring
	mtx  sync.Mutex

	TcpPortPrefix int
	UdpPortPrefix int
	IsTesting     bool
}

// NewIpRing creates and initializes a new IP ring buffer
func NewIpRing() *IpRing {
	r := ring.New(ringSize)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + ipStart
		r = r.Next()
	}
	return &IpRing{
		Ring:          r,
		TcpPortPrefix: 6114,
		UdpPortPrefix: 6113,
	}
}

// Reset resets the ring to its initial state
func (r *IpRing) Reset() {
	// Noop
}

// NextInt returns the next integer value from the ring
func (r *IpRing) NextInt() int {
	if r == nil {
		return 0
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()
	d := r.Ring.Value.(int)
	r.Ring = r.Ring.Next()
	return d
}

// NextAddr returns the next IP address and port numbers for TCP and UDP
func (r *IpRing) NextAddr() (ip net.IP, portTCP string, portUDP string, err error) {
	if r == nil {
		return nil, "", "", fmt.Errorf("ip ring is nil")
	}

	if !r.IsTesting {
		ip = net.IPv4(127, 0, 1, byte(r.NextInt()))
		return ip, "", "", nil
	}

	ip = net.ParseIP(localhost)
	next := r.NextInt()

	portTCP = fmt.Sprintf("%d%d", r.TcpPortPrefix, next)
	portUDP = fmt.Sprintf("%d%d", r.UdpPortPrefix, next)

	// Validate generated port numbers
	tcpPort := r.TcpPortPrefix*10 + next
	udpPort := r.UdpPortPrefix*10 + next
	if tcpPort > maxPortNumber || udpPort > maxPortNumber {
		return ip, "", "", fmt.Errorf("generated port numbers exceed maximum allowed value")
	}

	return ip, portTCP, portUDP, nil
}
