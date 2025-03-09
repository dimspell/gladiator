package p2p

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"

	"github.com/dimspell/gladiator/internal/backend/bsession"
)

// GameManager coordinates peer connections and game state, while Game represents
// an active game session with connected peers. The package uses WebRTC for direct
// communication between players and maintains IP address allocation for the network.
type GameManager struct {
	config  webrtc.Configuration
	session *bsession.Session

	Game *Game
}

// Reset clears the current game state for the session.
func (g *GameManager) Reset() {
	g.Game.Close()
	g.Game = nil
}

// CreatePeer sets up the peer connection channels for the given player. If a peer
// connection already exists for the player, it returns the existing peer. Otherwise,
// it creates a new peer connection and returns a new Peer instance.
//
// If the GameManager has no active game, it returns an error. If there is an error
// creating the new peer connection, it returns the error.
func (g *GameManager) CreatePeer(player wire.Player) (*Peer, error) {
	if g.Game == nil {
		return nil, fmt.Errorf("could not find mapping for user ID: %d", g.session.GetUserID())
	}

	if peer, found := g.Game.Peers[player.UserID]; found {
		slog.Debug("Reusing peer", "userId", player.ID())
		return peer, nil
	}

	peerConnection, err := webrtc.NewPeerConnection(g.config)
	if err != nil {
		return nil, err
	}

	isHost := g.Game.IsHost(player.UserID)
	isCurrentUser := g.Game.IsHost(g.session.UserID)

	peer, err := NewPeer(peerConnection, g.Game.IpRing, player.UserID, isCurrentUser, isHost)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{}, 1)
	peer.Connected = ch

	return peer, nil
}

func (g *GameManager) Host() (*Peer, bool) {
	if g.Game == nil {
		return nil, false
	}
	return g.GetPeer(g.Game.Host.UserID)
}

func (g *GameManager) SetHost(peer *Peer, newHost wire.Player) {
	if g.Game == nil {
		return
	}

	g.Game.mtx.Lock()
	g.Game.Host = newHost
	g.Game.mtx.Unlock()
}

// AddPeer adds a peer to the game.
func (g *GameManager) AddPeer(peer *Peer) {
	if g.Game == nil {
		return
	}

	g.Game.mtx.Lock()
	defer g.Game.mtx.Unlock()

	g.Game.Peers[peer.UserID] = peer
}

// GetPeer retrieves a peer by UserID.
func (g *GameManager) GetPeer(userId int64) (*Peer, bool) {
	if g.Game == nil {
		return nil, false
	}

	g.Game.mtx.Lock()
	defer g.Game.mtx.Unlock()

	peer, ok := g.Game.Peers[userId]
	return peer, ok
}

func (g *GameManager) RemovePeer(userId int64) {
	if g.Game == nil {
		return
	}

	g.Game.mtx.Lock()
	defer g.Game.mtx.Unlock()

	delete(g.Game.Peers, userId)
}

// Game represents a game room.
type Game struct {
	mtx sync.Mutex

	// Name of the game room
	ID string

	// Player who is the host of the game room
	Host wire.Player

	// A map of the players who are connected to the game room (except the current player) identified by user-id.
	Peers map[int64]*Peer

	// Controller to find the next free IP address
	IpRing *IpRing
}

// IsHost checks if the provided user is hosting the game.
func (g *Game) IsHost(userId int64) bool {
	return g.Host.UserID == userId
}

func (g *Game) Close() {
	g.closeAllConnections()
}

// closeAllConnections disconnects from all the peers.
func (g *Game) closeAllConnections() {
	if g == nil {
		return
	}
	for _, peer := range g.Peers {
		peer.Terminate()
	}
}
