package lobby

import (
	"context"
	"fmt"
	"github.com/coder/websocket"
	"log/slog"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/wire"
)

type Multiplayer struct {
	done context.CancelFunc

	// Sessions
	sessionMutex sync.RWMutex
	sessions     map[string]*UserSession

	// Presence chan UserSession
	Messages chan wire.Message
}

func NewMultiplayer(ctx context.Context) *Multiplayer {
	ctx, done := context.WithCancel(ctx)

	mp := &Multiplayer{
		sessions: make(map[string]*UserSession),
		Messages: make(chan wire.Message),
		done:     done,
	}

	go mp.Run(ctx)
	return mp
}

func (mp *Multiplayer) Close() { mp.done() }

func (mp *Multiplayer) Reset() {
	mp.forEachSession(func(userSession *UserSession) bool {
		_ = userSession.wsConn.CloseNow()
		return true
	})
	clear(mp.sessions)
	close(mp.Messages)
}

func (mp *Multiplayer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Received signal, closing the server")
			mp.Reset()
			return
		case msg, ok := <-mp.Messages:
			if !ok {
				return
			}
			mp.HandleIncomingMessage(ctx, msg)
		}
	}
}

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
// 		Messages: make(chan icesignal.Message),
// 	}
// 	s.SetChannel(channelName, c)
// 	go c.Run(ctx)
// 	return c
// 	// }
// }

// HandleIncomingMessage handles the incoming message pump by dispatching
// commands based on the message type.
func (mp *Multiplayer) HandleIncomingMessage(ctx context.Context, msg wire.Message) {
	slog.Debug("Received a signal message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	switch msg.Type {
	case wire.Chat:
		mp.BroadcastMessage(ctx, wire.Compose(wire.Chat, wire.Message{
			From:    msg.From,
			Content: msg.Content,
		}))
	case wire.RTCOffer, wire.RTCAnswer, wire.RTCICECandidate:
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

	// Expect the character info, then join and synchronise the state.
	if err := mp.HandleJoin(ctx, session); err != nil {
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
		_, m, err := wire.Decode(payload)
		if err != nil {
			return err
		}
		mp.Messages <- m
	}
}

func (mp *Multiplayer) ForwardRTCMessage(ctx context.Context, msg wire.Message) {
	slog.Debug("Forwarding RTC message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if user, ok := mp.GetUserSession(msg.To); ok {
		user.SendMessage(ctx, msg.Type, msg)
	}
}

// DebugState returns all information about the lobby.
func (mp *Multiplayer) DebugState() {
	fmt.Println("Connected players", len(mp.sessions))
	for key, session := range mp.sessions {
		fmt.Println(key, fmt.Sprintf("%#v", session.ToPlayer()))
	}
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
	et, m, err := wire.DecodeTyped[wire.User](payload)
	if err != nil {
		return err
	}
	if et != wire.Hello {
		return fmt.Errorf("inapprioprate event type")
	}

	session.User = m.Content

	session.Send(ctx, []byte{byte(wire.Welcome)})
	return nil
}

func (mp *Multiplayer) HandleJoin(ctx context.Context, session *UserSession) error {
	payload, err := session.ReadNext(ctx)
	if err != nil {
		return err
	}
	et, m, err := wire.DecodeTyped[wire.Player](payload)
	if err != nil {
		return err
	}
	if et != wire.Join {
		return fmt.Errorf("inapprioprate event type")
	}

	session.Character.CharacterID = m.Content.CharacterID
	session.Character.ClassType = m.Content.ClassType

	session.Send(ctx, []byte{byte(wire.Joined)})
	return nil
}

// SetPlayerConnected notifies the user has connected to the lobby.
func (mp *Multiplayer) SetPlayerConnected(session *UserSession) {
	players := mp.listSessions()
	mp.AddUserSession(session.UserID, session)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	session.SendMessage(ctx, wire.LobbyUsers, wire.Message{
		Type:    wire.LobbyUsers,
		To:      session.UserID,
		Content: players,
	})

	// Notify all the users
	mp.BroadcastMessage(ctx, wire.ComposeTyped(wire.Join, wire.MessageContent[wire.Player]{
		Type:    wire.Join,
		Content: session.ToPlayer(),
	}))
}

// SetPlayerDisconnected notifies the user has left the lobby.
func (mp *Multiplayer) SetPlayerDisconnected(session *UserSession) {
	// TODO: Close the socket
	session.wsConn.CloseNow()
	mp.DeleteUserSession(session.UserID)
	mp.BroadcastMessage(context.TODO(), wire.Compose(wire.Leave, wire.Message{Type: wire.Leave, From: session.UserID}))
}

// BroadcastMessage sends a message to all connected users.
func (mp *Multiplayer) BroadcastMessage(ctx context.Context, payload []byte) {
	mp.forEachSession(func(session *UserSession) bool {
		session.Send(ctx, payload)
		return true
	})
}

// GetUserSession is a thread-safe method to receive a session by ID.
func (mp *Multiplayer) GetUserSession(id string) (*UserSession, bool) {
	mp.sessionMutex.RLock()
	member, ok := mp.sessions[id]
	mp.sessionMutex.RUnlock()
	return member, ok
}

// AddUserSession is a thread-safe operation to add a session identified by ID.
func (mp *Multiplayer) AddUserSession(id string, session *UserSession) {
	if _, exists := mp.GetUserSession(id); exists {
		return
	}
	mp.sessionMutex.Lock()
	mp.sessions[id] = session
	mp.sessionMutex.Unlock()
}

// DeleteUserSession is a thread-safe operation to delete a session by ID.
func (mp *Multiplayer) DeleteUserSession(id string) {
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
func (mp *Multiplayer) listSessions() []wire.Player {
	mp.sessionMutex.RLock()
	defer mp.sessionMutex.RUnlock()

	list := make([]wire.Player, len(mp.sessions))
	i := 0
	for _, session := range mp.sessions {
		list[i] = session.ToPlayer()
		i++
	}
	return list
}
