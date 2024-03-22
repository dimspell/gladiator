package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"
)

var _ Proxy = (*Loopback)(nil)

type Loopback struct {
	Test bool

	GlobalProxyAddress string
	GlobalProxyConn    net.Conn

	mtx               sync.RWMutex
	HostIPAddress     net.IP
	ClientIPAddresses [4]net.IP
	MapUserToIP       map[string]net.IP
	MapIPIndexToUser  map[int]string

	CurrentPlayer string
	HostPlayer    string

	tcpDataCh chan Data
	udpDataCh chan Data
	closeCh   chan struct{}
}

func NewLoopback(masterProxyAddr string) (*Loopback, error) {
	p := &Loopback{
		GlobalProxyAddress: net.JoinHostPort("localhost", "6115"),

		ClientIPAddresses: [4]net.IP{
			net.IPv4(127, 0, 1, 1),
			net.IPv4(127, 0, 1, 2),
			net.IPv4(127, 0, 1, 3),
			net.IPv4(127, 0, 1, 4),
		},

		MapUserToIP:      make(map[string]net.IP),
		MapIPIndexToUser: make(map[int]string),
		tcpDataCh:        make(chan Data),
		udpDataCh:        make(chan Data),
	}
	p.HostIPAddress = p.ClientIPAddresses[0]

	return p, nil
}

// Create is used to start serving the traffic to the game host
func (p *Loopback) Create(_ string, hostUser string) (net.IP, error) {
	// p.Close()

	// todo: mutex please
	// p.mtx.Lock()
	// defer p.mtx.Unlock()
	hostIP := p.ClientIPAddresses[0]
	p.HostIPAddress = hostIP
	p.HostPlayer = hostUser

	p.CurrentPlayer = hostUser
	p.MapUserToIP[hostUser] = hostIP
	p.MapIPIndexToUser[0] = hostUser

	if err := p.connect(); err != nil {
		return nil, err
	}

	go func() {
		p.startTCP(context.TODO())
	}()

	return hostIP, nil
}

// Join is used to connect to TCP game host
func (p *Loopback) Join(gameId string, currentPlayer string, _ string) (net.IP, error) {
	// p.Close()

	if err := p.connect(); err != nil {
		return nil, err
	}

	// p.mtx.Lock()
	p.CurrentPlayer = currentPlayer
	// p.mtx.Unlock()

	go func() {
		p.startTCP(context.TODO())
	}()

	return p.GetHostIP(""), nil
}

// Exchange is used by UDP clients
func (p *Loopback) Exchange(gameId string, userId string, _ string) (net.IP, error) {
	udpIP, index, err := p.nextIP()
	if err != nil {
		return nil, err
	}

	p.CurrentPlayer = userId
	p.MapUserToIP[userId] = udpIP
	p.MapIPIndexToUser[index] = userId

	go func() {
		p.startUDP(context.TODO(), index, udpIP)
	}()

	return udpIP, nil
}

func (p *Loopback) nextIP() (net.IP, int, error) {
	if len(p.MapIPIndexToUser) > 4 {
		return nil, 0, fmt.Errorf("beta: ip-user map not cleaned up")
	}
	index := len(p.MapIPIndexToUser)
	ip := p.ClientIPAddresses[index]
	return ip, index, nil
}

func (p *Loopback) GetHostIP(_ string) net.IP {
	return p.HostIPAddress
}

func (p *Loopback) Close() {
	if p.closeCh != nil {
		close(p.closeCh)
	}

	p.GlobalProxyConn = nil
	clear(p.MapIPIndexToUser)
	clear(p.MapIPIndexToUser)
}
