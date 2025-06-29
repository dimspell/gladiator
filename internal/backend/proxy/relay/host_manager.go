package relay

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
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"golang.org/x/sync/errgroup"
)

type HostManager struct {
	// key: ip
	hosts map[string]*FakeHost
	// key: remoteID
	peerHosts map[string]*FakeHost

	// key: remoteID, value: localIP
	peerIPs map[string]string

	// reverse map - fakeLAN IP => remoteID
	ipToPeerID map[string]string
	mu         sync.Mutex

	ipPrefix net.IP
}

func NewManager(ipPrefix net.IP) *HostManager {
	return &HostManager{
		hosts:      make(map[string]*FakeHost),
		peerHosts:  make(map[string]*FakeHost),
		peerIPs:    make(map[string]string),
		ipToPeerID: make(map[string]string),
		ipPrefix:   ipPrefix,
	}
}

type FakeHost struct {
	IP       string
	LastSeen time.Time

	ProxyUDP redirect.Redirect
	ProxyTCP redirect.Redirect
	stopFunc context.CancelFunc
}

// Dynamic IP Allocator
func (hm *HostManager) assignIP(remoteID string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Already assigned?
	if ip, ok := hm.peerIPs[remoteID]; ok {
		return ip, nil
	}

	// Try from 127.0.0.2-127.0.0.254
	for i := 2; i < 255; i++ {
		ip := net.IPv4(127, 0, hm.ipPrefix[2], byte(i)).To4()
		ipAddr := ip.String()
		if _, ok := hm.ipToPeerID[ipAddr]; !ok {
			hm.peerIPs[remoteID] = ipAddr
			hm.ipToPeerID[ipAddr] = remoteID
			return ipAddr, nil
		}
	}
	return "", fmt.Errorf("no available IPs")
}

// StartDialHost adds a new dynamic joiner that dials our game client address
func (hm *HostManager) StartDialHost(
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

	if _, exists := hm.hosts[ipAddress]; exists {
		log.Printf("Already started guest IP %s\n", ipAddress)
		return nil, fmt.Errorf("host %s already running", ipAddress)
	}

	var err error
	var tcpProxy, udpProxy redirect.Redirect

	// TCP dialer to the local game server
	if realTCPPort > 0 {
		tcpProxy, err = redirect.DialTCP("127.0.0.1", strconv.Itoa(realTCPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to dial TCP: %w", err)
		}
	}

	// UDP dialer
	if realUDPPort > 0 {
		udpProxy, err = redirect.DialUDP("127.0.0.1", strconv.Itoa(realUDPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to dial UDP: %w", err)
		}
	}

	g, ctx := errgroup.WithContext(context.Background())
	ctx, cancel := context.WithCancel(ctx)

	host := &FakeHost{
		IP:       ipAddress,
		LastSeen: time.Now(),
		stopFunc: cancel,
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func(host *FakeHost) {
		g.Go(func() error {
			wg.Done()
			if tcpProxy == nil {
				return nil
			}
			return tcpProxy.Run(ctx, func(p []byte) (err error) {
				slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))

				host.LastSeen = time.Now()
				return onReceiveTCP(p)
			})
		})

		g.Go(func() error {
			wg.Done()
			return udpProxy.Run(ctx, func(p []byte) (err error) {
				if udpProxy == nil {
					return nil
				}

				slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))

				host.LastSeen = time.Now()
				return onReceiveUDP(p)
			})
		})

		wg.Done()
		if err := g.Wait(); err != nil {
			slog.Warn("UDP/TCP fake host failed", logging.Error(err))
			return
		}
	}(host)

	hm.hosts[ipAddress] = host
	hm.peerHosts[peerID] = host

	wg.Wait()
	return host, nil
}

// StartListenerHost starts a fake host on a loopback IP
func (hm *HostManager) StartListenerHost(
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

	if _, exists := hm.hosts[ipAddress]; exists {
		return nil, fmt.Errorf("host %s already running", ipAddress)
	}

	var err error
	var tcpProxy, udpProxy redirect.Redirect

	var wg sync.WaitGroup
	wg.Add(1)

	// TCP listener that mimics a peer in LAN
	if realTCPPort > 0 {
		tcpProxy, err = redirect.ListenTCP(ipAddress, strconv.Itoa(realTCPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on TCP: %w", err)
		}
		wg.Add(1)
	}

	// UDP listener
	if realUDPPort > 0 {
		udpProxy, err = redirect.ListenUDP(ipAddress, strconv.Itoa(realUDPPort))
		if err != nil {
			return nil, fmt.Errorf("failed to listen on UDP: %w", err)
		}
		wg.Add(1)
	}

	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)

	host := &FakeHost{
		IP:       ipAddress,
		LastSeen: time.Now(),
		stopFunc: cancel,
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	go func(host *FakeHost) {
		if tcpProxy != nil {
			g.Go(func() error {
				wg.Done()
				return tcpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[TCP] GameClient => Remote", "data", p, logging.PeerID(peerID))

					host.LastSeen = time.Now()
					return onReceiveTCP(p)
				})
			})
		}

		if udpProxy != nil {
			g.Go(func() error {
				wg.Done()
				return udpProxy.Run(ctx, func(p []byte) (err error) {
					slog.Debug("[UDP] GameClient => Remote", "data", p, logging.PeerID(peerID))

					host.LastSeen = time.Now()
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

	hm.hosts[ipAddress] = host
	hm.peerHosts[peerID] = host

	wg.Wait()
	return host, nil
}

func (hm *HostManager) RemoveByIP(ipAddrOrPrefix string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for ipAddress, host := range hm.hosts {
		if strings.HasPrefix(ipAddress, ipAddrOrPrefix) {
			hm.stopHost(host, ipAddress)
		}
	}
}

func (hm *HostManager) RemoveByRemoteID(remoteID string) {
	log.Printf("Cleaning up guest host for peer %s", remoteID)

	hm.mu.Lock()
	defer hm.mu.Unlock()

	ip, exists := hm.peerIPs[remoteID]
	if !exists {
		return
	}

	host, exists := hm.hosts[ip]
	if !exists {
		return
	}

	hm.stopHost(host, ip)
}

func (hm *HostManager) stopHost(host *FakeHost, ipAddress string) {
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
	remoteID, _ := hm.ipToPeerID[ipAddress]
	delete(hm.hosts, ipAddress)
	delete(hm.ipToPeerID, ipAddress)
	delete(hm.peerIPs, remoteID)
	delete(hm.peerHosts, remoteID)

	slog.Info("Fake host cleaned up", "ip", ipAddress)
}

// CleanupInactive does a cleanup of hosts that haven't been used in X seconds
// TODO: Unused
func (hm *HostManager) CleanupInactive(timeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now().Add(timeout)
	for ipAddress, host := range hm.hosts {
		if host.LastSeen.After(now) {
			slog.Info("Removing inactive host", "ip", ipAddress)
			hm.stopHost(host, ipAddress)
		}
	}
}
