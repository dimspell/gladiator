package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"golang.org/x/sync/errgroup"
)

type HostManager struct {
	mu       sync.Mutex
	IpPrefix net.IP

	// key: ip
	Hosts map[string]*FakeHost
	// key: remoteID
	PeerHosts map[string]*FakeHost

	// key: remoteID, value: localIP
	PeerIPs map[string]string

	// reverse map - fakeLAN IP => remoteID
	IPToPeerID map[string]string
}

func NewManager(ipPrefix net.IP) *HostManager {
	return &HostManager{
		IpPrefix:   ipPrefix,
		Hosts:      make(map[string]*FakeHost),
		PeerHosts:  make(map[string]*FakeHost),
		PeerIPs:    make(map[string]string),
		IPToPeerID: make(map[string]string),
	}
}

func (hm *HostManager) StopAll() {
	for _, host := range hm.Hosts {
		hm.StopHost(host)
	}

	hm.Hosts = make(map[string]*FakeHost)
	hm.PeerHosts = make(map[string]*FakeHost)
	hm.PeerIPs = make(map[string]string)
	hm.IPToPeerID = make(map[string]string)
}

// Dynamic IP Allocator
func (hm *HostManager) AssignIP(remoteID string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Already assigned?
	if ip, ok := hm.PeerIPs[remoteID]; ok {
		return ip, nil
	}

	// Try from 127.0.0.2-127.0.0.254
	for i := 2; i < 255; i++ {
		ip := net.IPv4(127, 0, hm.IpPrefix[2], byte(i)).To4()
		ipAddr := ip.String()
		if _, ok := hm.IPToPeerID[ipAddr]; !ok {
			hm.PeerIPs[remoteID] = ipAddr
			hm.IPToPeerID[ipAddr] = remoteID
			return ipAddr, nil
		}
	}
	return "", fmt.Errorf("no available IPs")
}

type FakeHost struct {
	Type       string
	PeerID     string
	AssignedIP string

	ProxyUDP Redirect
	ProxyTCP Redirect

	stopFunc context.CancelFunc
	closed   bool
}

// StartGuest adds a new dynamic joiner that dials our game client address
func (hm *HostManager) StartGuest(
	ctx context.Context,
	peerID string,
	assignedIP string,
	tcpPort, udpPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
	onHostDisconnect func(host *FakeHost, forced bool),
) (*FakeHost, error) {
	return hm.CreateFakeHost(
		ctx,
		"DIAL",
		peerID,
		assignedIP,
		&ProxySpec{
			LocalIP:   "127.0.0.1",
			Port:      tcpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return DialTCP(ipv4, port) },
			OnReceive: onReceiveTCP,
		},
		&ProxySpec{
			LocalIP:   "127.0.0.1",
			Port:      udpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return DialUDP(ipv4, port) },
			OnReceive: onReceiveUDP,
		},
		onHostDisconnect,
	)
}

// StartHost starts a fake host listening on a loopback IP
func (hm *HostManager) StartHost(
	ctx context.Context,
	peerID, assignedIP string,
	tcpPort, udpPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
	onHostDisconnect func(host *FakeHost, forced bool),
) (*FakeHost, error) {
	return hm.CreateFakeHost(
		ctx,
		"LISTEN",
		peerID,
		assignedIP,
		&ProxySpec{
			LocalIP:   assignedIP,
			Port:      tcpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return ListenTCP(ipv4, port) },
			OnReceive: onReceiveTCP,
		},
		&ProxySpec{
			LocalIP:   assignedIP,
			Port:      udpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return ListenUDP(ipv4, port) },
			OnReceive: onReceiveUDP,
		},
		onHostDisconnect,
	)
}

type ProxySpec struct {
	LocalIP   string
	Port      int
	Create    func(ipv4, port string) (Redirect, error)
	OnReceive func([]byte) error
}

func (hm *HostManager) CreateFakeHost(
	ctx context.Context,
	fakeHostType string,
	peerID string,
	assignedIP string,
	tcpParams *ProxySpec,
	udpParams *ProxySpec,
	onHostDisconnect func(host *FakeHost, forced bool),
) (*FakeHost, error) {
	if net.ParseIP(assignedIP).To4() == nil {
		return nil, fmt.Errorf("invalid IP address: %s", assignedIP)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.Hosts[assignedIP]; exists {
		return nil, fmt.Errorf("host %s already running", assignedIP)
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)

	host := &FakeHost{
		Type:       fakeHostType,
		PeerID:     peerID,
		AssignedIP: assignedIP,
		stopFunc:   cancel,
	}

	if tcpParams != nil && tcpParams.Port > 0 {
		tcpProxy, err := tcpParams.Create(tcpParams.LocalIP, strconv.Itoa(tcpParams.Port))
		if err != nil {
			return nil, err
		}
		host.ProxyTCP = tcpProxy
		g.Go(func() error {
			err := tcpProxy.Run(ctx, func(p []byte) (err error) {
				slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))
				return tcpParams.OnReceive(p)
			})
			slog.Debug("Closed TCP proxy", "error", err)
			return err
		})
	}
	if udpParams != nil && udpParams.Port > 0 {
		udpProxy, err := udpParams.Create(udpParams.LocalIP, strconv.Itoa(udpParams.Port))
		if err != nil {
			return nil, err
		}
		host.ProxyUDP = udpProxy
		g.Go(func() error {
			err := udpProxy.Run(ctx, func(p []byte) (err error) {
				slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))
				return udpParams.OnReceive(p)
			})
			slog.Debug("Closed UDP proxy", "error", err)
			return err
		})
	}

	go func(host *FakeHost) {
		err := g.Wait()
		if err != nil {
			slog.Warn("Shutting down the fake host", logging.Error(err), logging.PeerID(peerID), slog.String("type", fakeHostType), slog.String("assignedIP", assignedIP))
		}
		cancel()
		hm.StopHost(host)
		if onHostDisconnect != nil {
			onHostDisconnect(host, errors.Is(err, io.EOF))
		}
	}(host)

	hm.Hosts[assignedIP] = host
	hm.PeerHosts[peerID] = host

	return host, nil
}

func (hm *HostManager) SetHost(ip, peerID string, host *FakeHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.PeerIPs[peerID] = ip
	hm.IPToPeerID[ip] = peerID
	hm.Hosts[ip] = host
	hm.PeerHosts[peerID] = host
}

func (hm *HostManager) RemoveByIP(ipAddrOrPrefix string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for ipAddress, host := range hm.Hosts {
		if strings.HasPrefix(ipAddress, ipAddrOrPrefix) {
			hm.StopHost(host)
		}
	}
}

func (hm *HostManager) RemoveByRemoteID(remoteID string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	host, exists := hm.PeerHosts[remoteID]
	if !exists {
		slog.Debug("Cleaning up guest host - not exist", logging.PeerID(remoteID))
		return
	}

	slog.Debug("Cleaning up guest host - going to stop", logging.PeerID(remoteID))
	hm.StopHost(host)
}

func (hm *HostManager) StopHost(host *FakeHost) {
	if host.closed {
		return
	}

	// Trigger a stop
	host.stopFunc()

	// Close the connections
	if p := host.ProxyTCP; p != nil {
		_ = p.Close()
	}
	if p := host.ProxyUDP; p != nil {
		_ = p.Close()
	}

	// Remove from maps
	remoteID, _ := hm.IPToPeerID[host.AssignedIP]
	delete(hm.Hosts, host.AssignedIP)
	delete(hm.IPToPeerID, host.AssignedIP)
	delete(hm.PeerIPs, remoteID)
	delete(hm.PeerHosts, remoteID)
	host.closed = true

	slog.Info("Fake host cleaned up", "ip", host.AssignedIP)
}
