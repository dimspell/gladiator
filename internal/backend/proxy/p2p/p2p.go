package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/pion/webrtc/v4"
)

var _ proxy.Proxy = (*PeerToPeer)(nil)

// PeerToPeer implements the Proxy interface for WebRTC-based peer-to-peer connections.
// It manages game rooms, peer connections, and network addressing for multiplayer games
type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	WebRTCConfig   webrtc.Configuration
	NewTCPRedirect redirect.NewRedirect
	NewUDPRedirect redirect.NewRedirect
	SessionStore   *SessionStore
}

func NewPeerToPeer(iceServers ...webrtc.ICEServer) *PeerToPeer {
	config := webrtc.Configuration{}
	config.ICEServers = append(config.ICEServers, iceServers...)

	return &PeerToPeer{
		hostIPAddress:  net.IPv4(127, 0, 1, 2),
		WebRTCConfig:   config,
		NewTCPRedirect: redirect.NewTCPRedirect,
		NewUDPRedirect: redirect.NewUDPRedirect,
		SessionStore:   &SessionStore{sessions: make(map[*bsession.Session]*GameManager)},
	}
}

// CreateRoom creates a new game room and assigns the session as the host.
// Returns the assigned IP address for the host player
func (p *PeerToPeer) CreateRoom(params proxy.CreateParams, session *bsession.Session) (net.IP, error) {
	gameManager, ok := p.SessionStore.Get(session)
	if !ok {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	gameManager.Reset()

	ipAddr := net.IPv4(127, 0, 0, 1)
	hostPlayer := session.ToPlayer(ipAddr)

	gameRoom := &Game{
		ID:     params.GameID,
		Host:   hostPlayer,
		Peers:  map[int64]*Peer{}, // FIXME: Add size limit
		IpRing: NewIpRing(),
	}

	gameManager.Game = gameRoom

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(ctx context.Context, params proxy.HostParams, session *bsession.Session) error {
	gameManager, ok := p.SessionStore.sessions[session]
	if !ok {
		return fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}
	if gameManager.Game == nil || gameManager.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) GetHostIP(hostIpAddress net.IP, session *bsession.Session) net.IP {
	return p.hostIPAddress
}

func (p *PeerToPeer) SelectGame(params proxy.GameData, session *bsession.Session) error {
	gameManager, ok := p.SessionStore.Get(session)
	if !ok {
		return fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	gameManager.Reset()

	hostPlayer, err := params.FindHostUser()
	if err != nil {
		return err
	}

	gameRoom := &Game{
		ID:     params.Game.GameId,
		Host:   hostPlayer,
		Peers:  map[int64]*Peer{}, // FIXME: Add size limit
		IpRing: NewIpRing(),
	}

	for _, player := range params.ToWirePlayers() {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return err
		}

		isCurrentUser := session.GetUserID() == player.UserID
		isHostUser := gameRoom.Host.UserID == player.UserID

		peer, err := NewPeer(peerConnection,
			gameRoom.IpRing,
			player.UserID,
			isCurrentUser,
			isHostUser)
		if err != nil {
			return err
		}
		gameRoom.Peers[player.UserID] = peer

		// if !isCurrentUser {
		// if err := peer.setupPeerConnection(context.TODO(), session, player, false); err != nil {
		// 	return err
		// }
	}

	gameManager.Game = gameRoom

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	gameManager, ok := p.SessionStore.Get(session)
	if !ok {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}
	peer, ok := gameManager.GetPeer(params.UserID)
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(ctx context.Context, params proxy.JoinParams, session *bsession.Session) (net.IP, error) {
	ip := net.IPv4(127, 0, 0, 1)

	gameManager, ok := p.SessionStore.Get(session)
	if !ok || gameManager.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	peer := &Peer{
		UserID: session.GetUserID(),
		Addr:   &redirect.Addressing{IP: ip},
		Mode:   redirect.None,
	}
	gameManager.AddPeer(peer)

	for _, pr := range gameManager.Game.Peers {
		ch := make(chan struct{}, 1)
		pr.Connected = ch
	}

	return ip, nil
}

func (p *PeerToPeer) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	gameManager, ok := p.SessionStore.Get(session)
	if !ok || gameManager.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	peer, ok := gameManager.Game.Peers[params.UserID]
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
	}

	if peer.Connected == nil {
		return nil, fmt.Errorf("peer does not have a connection channel")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		slog.Error("timeout waiting for peer to connect", "user_id", params.UserID)
	case <-peer.Connected:
		slog.Debug("peer connected, user ID", "user_id", params.UserID)
	}

	return peer.Addr.IP, nil
}

// Close closes the connection for a session.
func (p *PeerToPeer) Close(session *bsession.Session) {
	gameManager, ok := p.SessionStore.Get(session)
	if !ok {
		return
	}

	gameManager.Reset()

	p.SessionStore.Delete(session)
}

func (p *PeerToPeer) NewWebSocketHandler(session *bsession.Session) proxy.MessageHandler {
	gameManager := &GameManager{
		session: session,
		config:  p.WebRTCConfig,
	}

	p.SessionStore.Add(session, gameManager)

	return &PeerToPeerMessageHandler{
		session.GetUserID(),
		session,
		gameManager,
		p.NewTCPRedirect,
		p.NewUDPRedirect,
		slog.With("user_id", session.GetUserID()),
	}
}

// SessionStore manages game managers for different sessions.
type SessionStore struct {
	sessions map[*bsession.Session]*GameManager
	mutex    sync.RWMutex
}

// Get retrieves a game manager for a session.
func (ss *SessionStore) Get(session *bsession.Session) (*GameManager, bool) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	gameManager, exists := ss.sessions[session]
	return gameManager, exists
}

func (ss *SessionStore) Add(session *bsession.Session, manager *GameManager) {
	ss.mutex.Lock()
	ss.sessions[session] = manager
	ss.mutex.Unlock()
}

func (ss *SessionStore) Delete(session *bsession.Session) {
	ss.mutex.Lock()
	delete(ss.sessions, session)
	ss.mutex.Unlock()
}
