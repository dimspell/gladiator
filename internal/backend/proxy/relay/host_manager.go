package relay

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/backend/redirect"
)

type HostManager struct {
	hosts      map[string]*FakeHost // key: ip
	peerIPs    map[string]string    // key: remoteID, value: localIP
	ipToPeerID map[string]string    // reverse map - fakeLAN IP => remoteID
	mu         sync.Mutex
}

func NewManager() *HostManager {
	return &HostManager{
		hosts:      make(map[string]*FakeHost),
		peerIPs:    make(map[string]string),
		ipToPeerID: make(map[string]string),
	}
}

type FakeHost struct {
	IP       string
	LastSeen time.Time
	StopChan chan struct{}

	ProxyUDP redirect.Redirect
	ProxyTCP redirect.Redirect
}

// func (hm *HostManager) GetHost(key string) *FakeHost {
// 	hm.mu.Lock()
// 	defer hm.mu.Unlock()
// 	return hm.hosts[key]
// }

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
		ip := fmt.Sprintf("127.0.0.%d", i)
		if _, ok := hm.ipToPeerID[ip]; !ok {
			hm.peerIPs[remoteID] = ip
			hm.ipToPeerID[ip] = remoteID
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available IPs")
}

// StartGuestHost adds new dynamic joiner
func (hm *HostManager) StartGuestHost(ipAddress string, realTCPPort, realUDPPort int) error {
	if net.ParseIP(ipAddress) == nil {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.hosts[ipAddress]; exists {
		log.Printf("Already started guest IP %s\n", ipAddress)
		return fmt.Errorf("host %s already running", ipAddress)
	}

	ctx := context.TODO()

	var err error
	var tcpProxy, udpProxy redirect.Redirect

	// TCP dialer to the local game server
	if realTCPPort > 0 {
		tcpProxy, err = redirect.DialTCP(ipAddress, strconv.Itoa(realTCPPort))
		if err != nil {
			return fmt.Errorf("failed to dial TCP: %w", err)
		}

		go tcpProxy.Run(ctx, func(p []byte) (err error) {
			log.Printf("[TCP]: TODO Received: %s", p)

			hm.hosts[ipAddress].LastSeen = time.Now()
			return nil
		})
	}

	// UDP dialer
	if realUDPPort > 0 {
		udpProxy, err = redirect.DialUDP(ipAddress, strconv.Itoa(realUDPPort))
		if err != nil {
			return fmt.Errorf("failed to dial UDP: %w", err)
		}
		go udpProxy.Run(ctx, func(p []byte) (err error) {
			log.Printf("[UDP]: TODO Received: %s", p)

			hm.hosts[ipAddress].LastSeen = time.Now()
			return nil
		})
	}

	// Save connection if needed
	hm.hosts[ipAddress] = &FakeHost{
		IP:       ipAddress,
		LastSeen: time.Now(),
		StopChan: make(chan struct{}), // FIXME: Unused
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}

	return nil
}

// StartHost starts a fake host on a loopback IP
func (hm *HostManager) StartHost(ipAddress string, realTCPPort, realUDPPort int) error {
	if net.ParseIP(ipAddress) == nil {
		return fmt.Errorf("invalid IP address: %s", ipAddress)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.hosts[ipAddress]; exists {
		return fmt.Errorf("host %s already running", ipAddress)
	}

	ctx := context.TODO()

	var err error
	var tcpProxy, udpProxy redirect.Redirect

	// TCP listener that mimics a peer in LAN
	if realTCPPort > 0 {
		tcpProxy, err = redirect.ListenTCP(ipAddress, strconv.Itoa(realTCPPort))
		if err != nil {
			return fmt.Errorf("failed to dial TCP: %w", err)
		}

		go tcpProxy.Run(ctx, func(p []byte) (err error) {
			log.Printf("[TCP]: TODO Received: %s", p)

			hm.hosts[ipAddress].LastSeen = time.Now()
			return nil
		})
	}

	// UDP listener
	if realUDPPort > 0 {
		udpProxy, err = redirect.ListenUDP(ipAddress, strconv.Itoa(realUDPPort))
		if err != nil {
			return fmt.Errorf("failed to dial UDP: %w", err)
		}
		go udpProxy.Run(ctx, func(p []byte) (err error) {
			log.Printf("[UDP]: TODO Received: %s", p)

			hm.hosts[ipAddress].LastSeen = time.Now()
			return nil
		})
	}

	hm.hosts[ipAddress] = &FakeHost{
		IP:       ipAddress,
		LastSeen: time.Now(),
		StopChan: make(chan struct{}), // FIXME: Unused
		ProxyTCP: tcpProxy,
		ProxyUDP: udpProxy,
	}
	return nil
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
	log.Printf("Cleaning up guest host for peer %s (%s)", remoteID)

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
	close(host.StopChan)

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

	log.Printf("Fake host at %s cleaned up", ipAddress)
}

// CleanupInactive does a cleanup of hosts that haven't been used in X seconds
// TODO: Unused
func (hm *HostManager) CleanupInactive(timeout time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now().Add(timeout)
	for ipAddress, host := range hm.hosts {
		if host.LastSeen.After(now) {
			log.Printf("Removing inactive host %s", ipAddress)
			hm.stopHost(host, ipAddress)
		}
	}
}
