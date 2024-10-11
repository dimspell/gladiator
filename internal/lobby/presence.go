package lobby

import (
	"context"

	"github.com/dimspell/gladiator/internal/icesignal"
)

const (
	// ErrLobbyNotFound error when lobby room was not found.
	ErrLobbyNotFound = iota

	// ErrLobbyAborted error when stopped while attempting to join.
	ErrLobbyAborted

	// ErrLobbyMissingPlayers when attempting to join empty lobby room.
	ErrLobbyMissingPlayers

	// ErrLobbyFull when no more players can join this lobby room.
	ErrLobbyFull
)

const (
	StateLobbyStarting = iota
	StateLobbyReady
	StateLobbyShuttingDown
	StateLobbyShutDown
	StatePlayerAlreadyConnected
	StatePlayerConnected
	StatePlayerDisconnected
)

// type LobbyRoom struct {
// 	Name     string
// 	Members  *Members
// 	Messages chan icesignal.Message
// }

// func (h *SignalServer) GetChannel(channelName string) (*Channel, bool) {
// 	h.RLock()
// 	channel, ok := h.Channels[channelName]
// 	h.RUnlock()
// 	return channel, ok
// }
//
// func (h *SignalServer) SetChannel(channelName string, channel *Channel) {
// 	h.Lock()
// 	h.Channels[channelName] = channel
// 	h.Unlock()
// }
//
// func (h *SignalServer) DeleteChannel(channelName string) {
// 	h.Lock()
// 	delete(h.Channels, channelName)
// 	h.Unlock()
// }
//
// func (s *SignalServer) Join(ctx context.Context, channelName string) *Channel {
// 	if existing, ok := s.GetChannel(channelName); ok {
// 		return existing
// 	}
//
// 	c := &Channel{
// 		Name:     channelName,
// 		Messages: make(chan icesignal2.Message),
// 	}
// 	s.SetChannel(channelName, c)
// 	go c.Run(ctx)
// 	return c
// 	// }
// }

// DebugState returns all information about the lobby.
func (mp *Multiplayer) DebugState() {

}

// CreateRoom creates new lobby room.
func (mp *Multiplayer) CreateRoom() {

}

// DestroyRoom deletes an existing lobby room.
func (mp *Multiplayer) DestroyRoom() {

}

// JoinRoom adds a player to an existing lobby room.
func (mp *Multiplayer) JoinRoom() {

}

// ListRooms lists & query all lobbies.
func (mp *Multiplayer) ListRooms() {

}

// GetRoom returns information about the LobbyRoom.
func (mp *Multiplayer) GetRoom() {

}

// SetRoomReady notifies the LobbyRoom that it can start accepting players.
func (mp *Multiplayer) SetRoomReady() {

}

// SetPlayerConnected notifies the user has connected to the lobby.
func (mp *Multiplayer) SetPlayerConnected(session *UserSession) {
	mp.addSession(session.UserID, session)
}

// SetPlayerDisconnected notifies the user has left the lobby.
func (mp *Multiplayer) SetPlayerDisconnected(session *UserSession) {
	// TODO: Close the socket
	session.Conn.CloseNow()
	mp.deleteSession(session.UserID)
	mp.BroadcastMessage(context.TODO(), compose(icesignal.Leave, icesignal.Message{Type: icesignal.Leave, From: session.UserID}))
}

// BroadcastMessage sends a message to all connected users.
func (mp *Multiplayer) BroadcastMessage(ctx context.Context, payload []byte) {
	mp.forEachSession(func(session *UserSession) bool {
		session.Send(ctx, payload)
		return true
	})
}

// getSession is a thread-safe method to receive a session by ID.
func (mp *Multiplayer) getSession(id string) (*UserSession, bool) {
	mp.sessionMutex.RLock()
	member, ok := mp.sessions[id]
	mp.sessionMutex.RUnlock()
	return member, ok
}

// addSession is a thread-safe operation to add a session identified by ID.
func (mp *Multiplayer) addSession(id string, session *UserSession) {
	if _, exists := mp.getSession(id); exists {
		return
	}
	mp.sessionMutex.Lock()
	mp.sessions[id] = session
	mp.sessionMutex.Unlock()
}

// deleteSession is a thread-safe operation to delete a session by ID.
func (mp *Multiplayer) deleteSession(id string) {
	mp.sessionMutex.Lock()
	delete(mp.sessions, id)
	mp.sessionMutex.Unlock()
}

// forEachSession is a thread-safe method to iterate over all sessions entries.
func (mp *Multiplayer) forEachSession(f func(session *UserSession) bool) {
	mp.sessionMutex.RLock()
	defer mp.sessionMutex.RUnlock()
	for _, member := range mp.sessions {
		if next := f(member); !next {
			return
		}
	}
}
