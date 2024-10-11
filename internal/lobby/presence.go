package lobby

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
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

// HandleIncomingMessage handles the incoming message pump by dispatching
// commands based on the message type.
func (mp *Multiplayer) HandleIncomingMessage(ctx context.Context, msg icesignal.Message) {
	slog.Debug("Received a signal message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	switch msg.Type {
	case icesignal.Chat:
		mp.BroadcastMessage(ctx, icesignal.Compose(icesignal.Chat, icesignal.Message{
			From:    msg.From,
			Content: msg.Content,
		}))
	case icesignal.RTCOffer, icesignal.RTCAnswer, icesignal.RTCICECandidate:
		mp.ForwardRTCMessage(ctx, msg)
	default:
		// Do nothing
	}
}

func (mp *Multiplayer) HandleSession(ctx context.Context, session *UserSession) error {
	// Expect the "hello" and send back "welcome" message.
	if err := mp.HandleHello(ctx, session); err != nil {
		return err
	}

	// Add user to the list of connected players.
	mp.SetPlayerConnected(session)

	// Remove the player
	defer mp.SetPlayerDisconnected(session)

	// Handle all the incoming messages.
	for {
		// Register that the user is still being active.
		session.LastSeen = time.Now().In(time.UTC)

		payload, err := session.ReadNext(ctx)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return err
		}
		if err != nil {
			slog.Error("Could not handle the message", "error", err)
			return err
		}

		// Enqueue message
		_, m, err := icesignal.Decode(payload)
		if err != nil {
			return err
		}
		mp.Messages <- m
	}
}

func (mp *Multiplayer) ForwardRTCMessage(ctx context.Context, msg icesignal.Message) {
	slog.Debug("Forwarding RTC message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if user, ok := mp.getSession(msg.To); ok {
		user.SendMessage(ctx, msg.Type, msg)
	}
}

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

func (mp *Multiplayer) HandleHello(ctx context.Context, session *UserSession) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	payload, err := session.ReadNext(ctx)
	if err != nil {
		return err
	}
	et, m, err := icesignal.DecodeTyped[icesignal.Player](payload)
	if err != nil {
		return err
	}
	if et != icesignal.Hello {
		return fmt.Errorf("inapprioprate event type")
	}

	session.Player = m.Content

	session.SendMessage(ctx, icesignal.Welcome, icesignal.Message{To: session.UserID})
	return nil
}

// SetPlayerConnected notifies the user has connected to the lobby.
func (mp *Multiplayer) SetPlayerConnected(session *UserSession) {
	mp.addSession(session.UserID, session)
	players := mp.listSessions()

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	session.SendMessage(ctx, icesignal.LobbyUsers, icesignal.Message{
		Type:    icesignal.LobbyUsers,
		To:      session.UserID,
		Content: players, // TODO: Map it to some readable form
	})

	mp.BroadcastMessage(ctx, icesignal.Compose(icesignal.Join, icesignal.Message{
		Type:    icesignal.Join,
		Content: session.UserID,
	}))
}

// SetPlayerDisconnected notifies the user has left the lobby.
func (mp *Multiplayer) SetPlayerDisconnected(session *UserSession) {
	// TODO: Close the socket
	session.wsConn.CloseNow()
	mp.deleteSession(session.UserID)
	mp.BroadcastMessage(context.TODO(), icesignal.Compose(icesignal.Leave, icesignal.Message{Type: icesignal.Leave, From: session.UserID}))
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

// listSession is a thread-safe method to retrieve the sessions list.
func (mp *Multiplayer) listSessions() []icesignal.Player {
	mp.sessionMutex.RLock()
	defer mp.sessionMutex.RUnlock()

	list := make([]icesignal.Player, len(mp.sessions))
	i := 0
	for _, session := range mp.sessions {
		list[i] = session.Player
		i++
	}
	return list
}
