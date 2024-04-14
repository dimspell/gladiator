package proxy

import (
	"container/ring"
	"fmt"
	"net"
	"sync"
)

var _ Proxy = (*Loopback)(nil)

// TODO: Not thread-safe. Implement RW-Mutexes

type IpRing *ring.Ring

func NewIpRing() IpRing {
	r := ring.New(3)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return r
}

func IPFromRing(r IpRing) net.IP {
	d := byte(r.Value.(int))
	return net.IPv4(127, 0, 1, d)
}

type Loopback struct {
	ProxyAddress string

	MapUserToIP      map[string]net.IP
	MapIPIndexToUser map[int]string

	mtx    sync.RWMutex
	Wire   *Wire
	IpRing *ring.Ring
}

func NewLoopback(masterProxyAddr string) (*Loopback, error) {
	masterProxyAddr = net.JoinHostPort("localhost", "6115")

	p := &Loopback{
		ProxyAddress:     masterProxyAddr,
		mtx:              sync.RWMutex{},
		MapUserToIP:      make(map[string]net.IP),
		MapIPIndexToUser: make(map[int]string),
	}
	p.Close()

	return p, nil
}

func (p *Loopback) init() {
	p.Wire = NewWire(p.ProxyAddress)
	p.IpRing = NewIpRing()
}

func (p *Loopback) Close() {
	p.Wire.Stop()
}

func (p *Loopback) Create(localIpAddress string, hostUser string) (net.IP, error) {
	creds := &Credentials{}
	host := &Player{
		IP:       p.GetHostIP(""),
		PlayerID: hostUser,
	}

	p.Close()
	p.init()
	if err := p.Wire.Start(creds, host, host); err != nil {
		return nil, err
	}

	return p.GetHostIP(hostUser), nil
}

func (p *Loopback) Join(gameId string, hostPlayer, currentPlayer string, _ string) (net.IP, error) {
	creds := &Credentials{}
	host := &Player{
		PlayerID: hostPlayer,
		IP:       p.GetHostIP(""),
	}
	me := &Player{
		PlayerID: currentPlayer,
		IP:       IPFromRing(p.IpRing),
	}

	p.Close()
	p.init()
	if err := p.Wire.Start(creds, host, me); err != nil {
		return nil, err
	}

	return p.GetHostIP(""), nil
}

func (p *Loopback) Exchange(gameId string, userId string, _ string) (net.IP, error) {
	if p.Wire == nil || !p.Wire.isConnected {
		return nil, fmt.Errorf("cannot exchange data over UDP - not connected")
	}

	udpIP := IPFromRing(p.IpRing)

	p.Wire.UDP()

	// p.Wire.startUDP()

	// p.CurrentPlayer = userId
	// p.MapUserToIP[userId] = udpIP
	// p.MapIPIndexToUser[index] = userId

	// go func() {
	// 	p.startUDP(context.TODO(), index, udpIP)
	// }()

	p.IpRing.Next()
	return udpIP, nil
}

// func (p *Loopback) nextIP() (net.IP, int, error) {
// 	if len(p.MapIPIndexToUser) > 4 {
// 		return nil, 0, fmt.Errorf("ip-user map has not been cleaned up")
// 	}
// 	index := len(p.MapIPIndexToUser)
// 	ip := p.ClientIPAddresses[index]
// 	return ip, index, nil
// }

func (p *Loopback) GetHostIP(_ string) net.IP {
	return net.IPv4(127, 0, 1, 1)
}
