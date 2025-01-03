package proxy

import (
	"sync"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
)

type SessionStore struct {
	sessions map[*bsession.Session]*SessionMapping
	mutex    sync.RWMutex
}

// SessionMapping maps sessions to their peers.
type SessionMapping struct {
	IpRing *IpRing
	Game   *GameRoom
	Peers  map[string]*Peer
}

func (ps *SessionStore) getOrCreatePeer(session *bsession.Session, player wire.Player) (*Peer, error) {
	mapping, ok := ps.sessions[session]
	if ok {
		peer, found := mapping.Peers[player.ID()]
		if found {
			return peer, nil
		}
	}

	gameRoom := mapping.Game
	gameRoom.SetPlayer(player)

	isHost := gameRoom.Host.UserID == player.UserID
	isCurrentUser := gameRoom.Host.UserID == session.UserID

	return NewPeer(mapping.IpRing, player.ID(), isCurrentUser, isHost)
}

func (ps *SessionStore) GetPeer(session *bsession.Session, userId string) (*Peer, bool) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	mapping, ok := ps.sessions[session]
	if !ok {
		return nil, false
	}
	peer, ok := mapping.Peers[userId]
	if !ok {
		return nil, false
	}
	return peer, true
}

func (ps *SessionStore) AddPeer(session *bsession.Session, peer *Peer) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	mapping, ok := ps.sessions[session]
	if !ok {
		return
	}
	mapping.Peers[peer.UserID] = peer
}

func (ps *SessionStore) RemovePeer(session *bsession.Session, userId string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	mapping, ok := ps.sessions[session]
	if !ok {
		return
	}
	delete(mapping.Peers, userId)
	mapping.Game.DeletePlayer(userId)
}
