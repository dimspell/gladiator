package redirect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"golang.org/x/sync/errgroup"
)

// ErrNotHost is returned by StartGuest when the configured IsHost gate reports
// that this backend is not the current room host. Only the host may open guest
// dialers toward other peers, so a non-host StartGuest is refused rather than
// creating noise and IP conflicts (the §8.1 pain point 5 bug class).
var ErrNotHost = errors.New("redirect: StartGuest refused: this peer is not the current host")

// ProxyKind and ProxyProtocol are type-safe enums for proxy creation.
type ProxyKind string
type ProxyProtocol string

const (
	KindDial   ProxyKind     = "dial"
	KindListen ProxyKind     = "listen"
	ProtoTCP   ProxyProtocol = "tcp"
	ProtoUDP   ProxyProtocol = "udp"
)

// ReceiveFunc is a callback for received data.
type ReceiveFunc func([]byte) error

// ProxySpec describes how to create a proxy for a FakeHost.
type ProxySpec struct {
	LocalIP   string
	Port      int
	Kind      ProxyKind
	Protocol  ProxyProtocol
	OnReceive ReceiveFunc
}

// FakeHost represents a running proxy host.
type FakeHost struct {
	Type       string
	PeerID     string
	AssignedIP string

	ProxyUDP Redirect
	ProxyTCP Redirect

	stopFunc context.CancelFunc
	closed   bool
	once     sync.Once
}

// HostManager manages FakeHosts and their proxies.
type HostManager struct {
	mu       sync.Mutex
	IPPrefix net.IP

	Hosts      map[string]*FakeHost // key: ip
	PeerHosts  map[string]*FakeHost // key: remoteID
	PeerIPs    map[string]string    // key: remoteID, value: localIP
	IPToPeerID map[string]string    // reverse map - fakeLAN IP => remoteID

	ProxyFactory ProxyFactory
	Logger       *slog.Logger

	// IsHost, when non-nil, gates StartGuest: a guest dialer is only created
	// when IsHost() reports true. This encodes the "StartGuest only when host"
	// rule inside the redirect layer so no caller can bypass it. Nil means no
	// gate (backward compatible; every caller bears the responsibility).
	IsHost func() bool
}

// NewManager creates a new HostManager with optional ProxyFactory, Logger, and Clock.
func NewManager(opts ...func(*HostManager)) *HostManager {
	hm := &HostManager{
		IPPrefix:     net.IPv4(127, 0, 0, 1),
		Hosts:        make(map[string]*FakeHost),
		PeerHosts:    make(map[string]*FakeHost),
		PeerIPs:      make(map[string]string),
		IPToPeerID:   make(map[string]string),
		ProxyFactory: &DefaultProxyFactory{},
		Logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(hm)
	}
	return hm
}

// WithProxyFactory allows injection of a custom proxy creation logic for testing.
func WithProxyFactory(factory ProxyFactory) func(*HostManager) {
	return func(hm *HostManager) { hm.ProxyFactory = factory }
}

func WithIPPrefix(ipPrefix net.IP) func(*HostManager) {
	return func(hm *HostManager) { hm.IPPrefix = ipPrefix }
}

// WithLogger allows injection of a custom logger for testing.
func WithLogger(logger *slog.Logger) func(*HostManager) {
	return func(hm *HostManager) { hm.Logger = logger }
}

// WithDisabledLogger disables logging.
func WithDisabledLogger() func(*HostManager) {
	return func(hm *HostManager) {
		hm.Logger = logger.NewDiscardLogger()
	}
}

// WithIsHost injects the predicate that decides whether this backend is the
// current room host. When non-nil and it returns false, StartGuest becomes a
// no-op that returns ErrNotHost, so only the host ever opens guest dialers
// toward other peers. This is the redirect-layer encoding of the
// "StartGuest only when host" rule.
func WithIsHost(isHost func() bool) func(*HostManager) {
	return func(hm *HostManager) { hm.IsHost = isHost }
}

// StopAll stops and removes all hosts.
func (hm *HostManager) StopAll() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	for _, host := range hm.Hosts {
		hm.stopHostLocked(host)
	}
	hm.Hosts = make(map[string]*FakeHost)
	hm.PeerHosts = make(map[string]*FakeHost)
	hm.PeerIPs = make(map[string]string)
	hm.IPToPeerID = make(map[string]string)
}

// AssignIP allocates a new IP for a remoteID, or returns the existing one.
func (hm *HostManager) AssignIP(remoteID string) (string, error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Already assigned?
	if ip, ok := hm.PeerIPs[remoteID]; ok {
		return ip, nil
	}

	// Try from 127.0.0.2-127.0.0.254 (or the equivalent range under a custom prefix)
	for i := 2; i < 255; i++ {
		base := hm.IPPrefix.To4()
		ip := net.IPv4(base[0], base[1], base[2], byte(i)).To4()
		ipAddr := ip.String()
		if _, ok := hm.IPToPeerID[ipAddr]; !ok {
			hm.PeerIPs[remoteID] = ipAddr
			hm.IPToPeerID[ipAddr] = remoteID
			return ipAddr, nil
		}
	}
	return "", fmt.Errorf("no available IPs")
}

// StartGuest adds a new dynamic joiner that dials our game client address.
func (hm *HostManager) StartGuest(
	ctx context.Context,
	peerID string,
	assignedIP string,
	tcpPort, udpPort int,
	onReceiveTCP, onReceiveUDP func([]byte) error,
	onHostDisconnect func(host *FakeHost, forced bool),
) (*FakeHost, error) {
	// Only the current host may open guest dialers toward other peers. This is
	// the authoritative guard: even if a caller forgets to check host status,
	// StartGuest refuses to create dialers as a non-host peer.
	if hm.IsHost != nil && !hm.IsHost() {
		hm.Logger.Debug("StartGuest refused: only the current host may create guest dialers", logging.PeerID(peerID))
		return nil, ErrNotHost
	}
	return hm.CreateFakeHost(
		ctx,
		"DIAL",
		peerID,
		assignedIP,
		&ProxySpec{
			LocalIP:  "127.0.0.1",
			Port:     tcpPort,
			Kind:     KindDial,
			Protocol: ProtoTCP,
			OnReceive: func(data []byte) error {
				hm.Logger.Debug("[TCP] GameClient => Remote", "data", data, logging.PeerID(peerID))
				return onReceiveTCP(data)
			},
		},
		&ProxySpec{
			LocalIP:  "127.0.0.1",
			Port:     udpPort,
			Kind:     KindDial,
			Protocol: ProtoUDP,
			OnReceive: func(data []byte) error {
				hm.Logger.Debug("[UDP] GameClient => Remote", "data", data, logging.PeerID(peerID))
				return onReceiveUDP(data)
			},
		},
		onHostDisconnect,
	)
}

// StartHost starts a fake host listening on a loopback IP.
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
			LocalIP:  assignedIP,
			Port:     tcpPort,
			Kind:     KindListen,
			Protocol: ProtoTCP,
			OnReceive: func(data []byte) error {
				hm.Logger.Debug("[TCP] GameClient => Remote", "data", data, logging.PeerID(peerID))
				return onReceiveTCP(data)
			},
		},
		&ProxySpec{
			LocalIP:  assignedIP,
			Port:     udpPort,
			Kind:     KindListen,
			Protocol: ProtoUDP,
			OnReceive: func(data []byte) error {
				hm.Logger.Debug("[UDP] GameClient => Remote", "data", data, logging.PeerID(peerID))
				return onReceiveUDP(data)
			},
		},
		onHostDisconnect,
	)
}

// CreateFakeHost creates and starts a FakeHost with the given proxy specs.
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

	var createdTCP bool

	if tcpParams != nil && tcpParams.Port > 0 {
		tcpProxy, err := hm.createProxy(tcpParams)
		if err != nil {
			cancel()
			return nil, err
		}
		host.ProxyTCP = tcpProxy
		createdTCP = tcpProxy != nil
		g.Go(func() error {
			if tcpProxy != nil {
				err := tcpProxy.Run(ctx)
				hm.Logger.Debug("Closed TCP proxy", "error", err)
				return err
			}
			return nil
		})
	}
	if udpParams != nil && udpParams.Port > 0 {
		udpProxy, err := hm.createProxy(udpParams)
		if err != nil {
			if createdTCP && host.ProxyTCP != nil {
				_ = host.ProxyTCP.Close()
			}
			cancel()
			return nil, err
		}
		host.ProxyUDP = udpProxy
		g.Go(func() error {
			if udpProxy != nil {
				err := udpProxy.Run(ctx)
				hm.Logger.Debug("Closed UDP proxy", "error", err)
				return err
			}
			return nil
		})
	}

	go func(host *FakeHost) {
		err := g.Wait()
		if err != nil {
			hm.Logger.Warn("Shutting down the fake host", logging.Error(err), logging.PeerID(peerID), slog.String("type", fakeHostType), slog.String("assignedIP", assignedIP))
		}
		cancel()
		hm.StopHost(host)
		if onHostDisconnect != nil {
			onHostDisconnect(host, err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) && !errors.Is(err, net.ErrClosed))
		}
	}(host)

	hm.Hosts[assignedIP] = host
	hm.PeerHosts[peerID] = host

	return host, nil
}

// createProxy creates a proxy based on the spec.
func (hm *HostManager) createProxy(spec *ProxySpec) (Redirect, error) {
	if spec == nil || spec.Port <= 0 {
		return nil, nil
	}
	ip := spec.LocalIP
	port := strconv.Itoa(spec.Port)
	var proxy Redirect
	var err error
	switch {
	case spec.Kind == KindDial && spec.Protocol == ProtoTCP:
		proxy, err = hm.ProxyFactory.NewDialTCP(ip, port, spec.OnReceive)
	case spec.Kind == KindDial && spec.Protocol == ProtoUDP:
		proxy, err = hm.ProxyFactory.NewDialUDP(ip, port, spec.OnReceive)
	case spec.Kind == KindListen && spec.Protocol == ProtoTCP:
		proxy, err = hm.ProxyFactory.NewListenerTCP(ip, port, spec.OnReceive)
	case spec.Kind == KindListen && spec.Protocol == ProtoUDP:
		proxy, err = hm.ProxyFactory.NewListenerUDP(ip, port, spec.OnReceive)
	default:
		err = fmt.Errorf("unknown proxy kind/protocol: %s/%s", spec.Kind, spec.Protocol)
	}
	return proxy, err
}

// SetHost sets a host in all maps.
func (hm *HostManager) SetHost(ip, peerID string, host *FakeHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.PeerIPs[peerID] = ip
	hm.IPToPeerID[ip] = peerID
	hm.Hosts[ip] = host
	hm.PeerHosts[peerID] = host
}

func (hm *HostManager) RemoveByIP(ipAddr string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for ipAddress, host := range hm.Hosts {
		if ipAddress == ipAddr {
			hm.stopHostLocked(host)
		}
	}
}

// RemoveByRemoteID removes a host by remoteID. Returns true if removed.
func (hm *HostManager) RemoveByRemoteID(remoteID string) bool {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	host, exists := hm.PeerHosts[remoteID]
	if !exists {
		hm.Logger.Debug("Cleaning up guest host - not exist", logging.PeerID(remoteID))
		return false
	}
	hm.Logger.Debug("Cleaning up guest host - going to stop", logging.PeerID(remoteID))
	hm.stopHostLocked(host)
	return true
}

// StopHost stops and removes a host safely.
func (hm *HostManager) StopHost(host *FakeHost) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.stopHostLocked(host)
}

// stopHostLocked stops a host (must be called with hm.mu held).
func (hm *HostManager) stopHostLocked(host *FakeHost) {
	if host == nil {
		return
	}
	host.once.Do(func() {
		host.closed = true
		if host.stopFunc != nil {
			host.stopFunc()
		}
		if host.ProxyTCP != nil {
			_ = host.ProxyTCP.Close()
		}
		if host.ProxyUDP != nil {
			_ = host.ProxyUDP.Close()
		}

		// Remove from maps
		remoteID := hm.IPToPeerID[host.AssignedIP]
		delete(hm.Hosts, host.AssignedIP)
		delete(hm.IPToPeerID, host.AssignedIP)
		delete(hm.PeerIPs, remoteID)
		delete(hm.PeerHosts, remoteID)
	})
}

// GetHostByIP returns a host by IP.
func (hm *HostManager) GetHostByIP(ip string) (*FakeHost, bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	host, ok := hm.Hosts[ip]
	return host, ok
}

// GetPeerHost returns a host by peerID.
func (hm *HostManager) GetPeerHost(peerID string) (*FakeHost, bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	host, ok := hm.PeerHosts[peerID]
	return host, ok
}

// GetPeerIP returns the assigned loopback IP for a remoteID.
func (hm *HostManager) GetPeerIP(remoteID string) (string, bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	ip, ok := hm.PeerIPs[remoteID]
	return ip, ok
}

// ForEachPeerHost iterates over all peer hosts under the lock. Returning false
// from fn stops iteration early.
func (hm *HostManager) ForEachPeerHost(fn func(peerID string, host *FakeHost) bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	for id, host := range hm.PeerHosts {
		if !fn(id, host) {
			break
		}
	}
}

// Len returns the number of IP-keyed hosts, peer-keyed hosts, and assigned
// peer IPs, all read atomically under the lock.
func (hm *HostManager) Len() (hosts, peerHosts, peerIPs int) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hosts, peerHosts, peerIPs = len(hm.Hosts), len(hm.PeerHosts), len(hm.PeerIPs)
	return
}

// ProxyFactory allows injection of custom proxy creation logic for testing.
type ProxyFactory interface {
	NewDialTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error)
	NewDialUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error)
	NewListenerTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error)
	NewListenerUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error)
}

// DefaultProxyFactory uses the real network constructors.
type DefaultProxyFactory struct{}

func (f *DefaultProxyFactory) NewDialTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return NewDialTCP(ip, port, onReceive)
}
func (f *DefaultProxyFactory) NewDialUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return NewDialUDP(ip, port, onReceive)
}
func (f *DefaultProxyFactory) NewListenerTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return NewListenerTCP(ip, port, onReceive)
}
func (f *DefaultProxyFactory) NewListenerUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return NewListenerUDP(ip, port, onReceive)
}
