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
	tcpPort, udpPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
) (*FakeHost, error) {
	return hm.CreateFakeHost(
		ctx,
		"DIAL",
		peerID,
		ipAddress,
		&ProxyParams{
			IPAddress: "127.0.0.1",
			Port:      tcpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return DialTCP(ipv4, port) },
			OnReceive: onReceiveTCP,
		},
		&ProxyParams{
			IPAddress: "127.0.0.1",
			Port:      udpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return DialUDP(ipv4, port) },
			OnReceive: onReceiveUDP,
		},
	)
}

// StartHost starts a fake host listening on a loopback IP
func (hm *HostManager) StartHost(
	ctx context.Context,
	peerID, ipAddress string,
	tcpPort, udpPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
) (*FakeHost, error) {
	return hm.CreateFakeHost(
		ctx,
		"LISTEN",
		peerID,
		ipAddress,
		&ProxyParams{
			IPAddress: ipAddress,
			Port:      tcpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return ListenTCP(ipv4, port) },
			OnReceive: onReceiveTCP,
		},
		&ProxyParams{
			IPAddress: ipAddress,
			Port:      udpPort,
			Create:    func(ipv4, port string) (Redirect, error) { return ListenUDP(ipv4, port) },
			OnReceive: onReceiveUDP,
		},
	)
}

type ProxyParams struct {
	IPAddress string
	Port      int
	Create    func(ipv4, port string) (Redirect, error)
	OnReceive func([]byte) error
}

func (hm *HostManager) CreateFakeHost(
	ctx context.Context,
	fakeHostType string,
	peerID string,
	ipAddress string,
	tcpParams *ProxyParams,
	udpParams *ProxyParams,
) (*FakeHost, error) {
	if net.ParseIP(ipAddress).To4() == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.Hosts[ipAddress]; exists {
		return nil, fmt.Errorf("host %s already running", ipAddress)
	}

	var err error
	var tcpProxy, udpProxy Redirect

	if tcpParams != nil && tcpParams.Port > 0 {
		tcpProxy, err = tcpParams.Create(tcpParams.IPAddress, strconv.Itoa(tcpParams.Port))
		if err != nil {
			return nil, err
		}
	}
	if udpParams != nil && udpParams.Port > 0 {
		udpProxy, err = udpParams.Create(udpParams.IPAddress, strconv.Itoa(udpParams.Port))
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)

	host := &FakeHost{
		Type:     fakeHostType,
		IP:       ipAddress,
		stopFunc: cancel,
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(host *FakeHost, wg *sync.WaitGroup) {
		if tcpProxy != nil {
			g.Go(func() error {
				err := tcpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return tcpParams.OnReceive(p)
				})
				slog.Debug("Closed TCP proxy", "error", err)
				return err
			})
		}
		if udpProxy != nil {
			g.Go(func() error {
				err := udpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))
					return udpParams.OnReceive(p)
				})
				slog.Debug("Closed UDP proxy", "error", err)
				return err
			})
		}

		wg.Done()
		if err := g.Wait(); err != nil {
			slog.Warn("UDP/TCP fake host failed", logging.Error(err))
			cancel()
			host.Dead = true
			return
		}
	}(host, wg)

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
