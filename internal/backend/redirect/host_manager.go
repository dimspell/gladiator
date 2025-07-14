package redirect

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

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
	for ipAddress, host := range hm.Hosts {
		hm.StopHost(host, ipAddress)
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
	Type string
	IP   string

	ProxyUDP Redirect
	ProxyTCP Redirect
	stopFunc context.CancelFunc
}

// StartGuest adds a new dynamic joiner that dials our game client address
func (hm *HostManager) StartGuest(
	ctx context.Context,
	peerID string,
	ipAddress string,
	realTCPPort, realUDPPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
) (*FakeHost, error) {
	if net.ParseIP(ipAddress) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.Hosts[ipAddress]; exists {
		log.Printf("Already started guest IP %s\n", ipAddress)
		return nil, fmt.Errorf("host %s already running", ipAddress)
	}

	var err error
	var tcpProxy, udpProxy Redirect

	if realTCPPort > 0 {
		tcpProxy, err = DialTCP("127.0.0.1", strconv.Itoa(realTCPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to dial TCP: %w", err)
		}
	}
	if realUDPPort > 0 {
		udpProxy, err = DialUDP("127.0.0.1", strconv.Itoa(realUDPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to dial UDP: %w", err)
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	host := &FakeHost{
		Type:     "DIAL",
		IP:       ipAddress,
		stopFunc: cancel,
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(host *FakeHost, wg *sync.WaitGroup) {
		if tcpProxy == nil {
			g.Go(func() error {
				return tcpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return onReceiveTCP(p)
				})
			})
		}
		if udpProxy != nil {
			g.Go(func() error {
				return udpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return onReceiveUDP(p)
				})
			})
		}

		wg.Done()
		if err := g.Wait(); err != nil {
			slog.Warn("UDP/TCP fake host failed", logging.Error(err))
			return
		}
	}(host, wg)

	hm.Hosts[ipAddress] = host
	hm.PeerHosts[peerID] = host

	wg.Wait()
	return host, nil
}

// StartHost starts a fake host listening on a loopback IP
func (hm *HostManager) StartHost(
	ctx context.Context,
	peerID string,
	ipAddress string,
	realTCPPort, realUDPPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
	livenessProbe func() error,
) (*FakeHost, error) {
	if net.ParseIP(ipAddress) == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.Hosts[ipAddress]; exists {
		return nil, fmt.Errorf("host %s already running", ipAddress)
	}

	var err error
	var tcpProxy, udpProxy Redirect

	if realTCPPort > 0 {
		tcpProxy, err = ListenTCP(ipAddress, strconv.Itoa(realTCPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on TCP: %w", err)
		}
	}
	if realUDPPort > 0 {
		udpProxy, err = ListenUDP(ipAddress, strconv.Itoa(realUDPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on UDP: %w", err)
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	host := &FakeHost{
		Type:     "LISTEN",
		IP:       ipAddress,
		stopFunc: cancel,
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func(host *FakeHost) {
		if tcpProxy != nil {
			g.Go(func() error {
				return tcpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return onReceiveTCP(p)
				})
			})
		}
		if udpProxy != nil {
			g.Go(func() error {
				return udpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return onReceiveUDP(p)
				})
			})
		}

		wg.Done()
		if err := g.Wait(); err != nil {
			slog.Warn("UDP/TCP fake host failed", logging.Error(err))
			return
		}
	}(host)

	hm.Hosts[ipAddress] = host
	hm.PeerHosts[peerID] = host

	wg.Wait()
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
			hm.StopHost(host, ipAddress)
		}
	}
}

func (hm *HostManager) RemoveByRemoteID(remoteID string) {
	log.Printf("Cleaning up guest host for peer %s", remoteID)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	ip, exists := hm.PeerIPs[remoteID]
	if !exists {
		return
	}

	host, exists := hm.Hosts[ip]
	if !exists {
		return
	}

	hm.StopHost(host, ip)
}

func (hm *HostManager) StopHost(host *FakeHost, ipAddress string) {
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
	remoteID, _ := hm.IPToPeerID[ipAddress]
	delete(hm.Hosts, ipAddress)
	delete(hm.IPToPeerID, ipAddress)
	delete(hm.PeerIPs, remoteID)
	delete(hm.PeerHosts, remoteID)

	slog.Info("Fake host cleaned up", "ip", ipAddress)
}

func (hm *HostManager) CleanupInactive(timeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// now := time.Now().Add(timeout)
	// for ipAddress, host := range hm.Hosts {
	// 	if host.LastSeen.After(now) {
	// 		slog.Info("Removing inactive host", "ip", ipAddress)
	// 		hm.StopHost(host, ipAddress)
	// 	}
	// }
}
