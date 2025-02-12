package p2p

import (
	"sync"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

type SessionStore struct {
	sessions map[*bsession.Session]*SessionMapping
	mutex    sync.RWMutex
}

func (ss *SessionStore) GetSession(session *bsession.Session) (*SessionMapping, bool) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	mapping, exists := ss.sessions[session]
	return mapping, exists
}

func (ss *SessionStore) SetSession(session *bsession.Session, mapping *SessionMapping) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	ss.sessions[session] = mapping
}

func (ss *SessionStore) DeleteSession(session *bsession.Session) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	delete(ss.sessions, session)
}

// SessionMapping maps sessions to their peers.
type SessionMapping struct {
	IpRing *IpRing
	Game   *proxy.GameRoom
	Peers  map[int64]*Peer
}

func (ss *SessionStore) GetPeer(session *bsession.Session, userId int64) (*Peer, bool) {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	mapping, ok := ss.sessions[session]
	if !ok {
		return nil, false
	}
	peer, ok := mapping.Peers[userId]
	if !ok {
		return nil, false
	}
	return peer, true
}

func (ss *SessionStore) AddPeer(session *bsession.Session, peer *Peer) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	mapping, ok := ss.sessions[session]
	if !ok {
		return
	}
	mapping.Peers[peer.UserID] = peer
}

func (ss *SessionStore) RemovePeer(session *bsession.Session, userId int64) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	mapping, ok := ss.sessions[session]
	if !ok {
		return
	}

	// peer, found := mapping.Peers[userId]

	delete(mapping.Peers, userId)
	mapping.Game.DeletePlayer(userId)
}
