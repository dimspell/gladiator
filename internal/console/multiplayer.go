package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/wire"
)

type Multiplayer struct {
	done context.CancelFunc

	// Presence in lobby
	sessionMutex sync.RWMutex
	sessions     map[int64]*UserSession

	Messages chan wire.Message

	// Game rooms
	roomsMutex sync.RWMutex
	Rooms      map[string]*GameRoom
}

func NewMultiplayer() *Multiplayer {
	mp := &Multiplayer{
		sessions: make(map[int64]*UserSession),
		Rooms:    make(map[string]*GameRoom),
		Messages: make(chan wire.Message),
	}
	return mp
}

func (mp *Multiplayer) Stop() { mp.done() }

func (mp *Multiplayer) Reset() {
	mp.forEachSession(func(userSession *UserSession) bool {
		_ = userSession.wsConn.CloseNow()
		return true
	})
	clear(mp.sessions)
	close(mp.Messages)
	clear(mp.Rooms)
}

func (mp *Multiplayer) Run(ctx context.Context) {
	ctx, done := context.WithCancel(ctx)
	mp.done = done

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
	case wire.SetRoomReady:
		mp.SetRoomReady(ctx, msg)
	case wire.LeaveRoom:
		mp.HandleLeaveRoom(ctx, msg)
	default:
		// Do nothing but log the event type
		slog.Error("Unhandled event type", "type", msg.Type.String())
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
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			switch state := websocket.CloseStatus(err); state {
			case -1:
				// connection reset by peer
				return nil
			case websocket.StatusNormalClosure:
				slog.Debug("Closing because of", "error", err)
				return err
			default:
				slog.Error("Could not handle the message", "error", err)
				return err
			}
		}

		// Enqueue message
		_, m, err := wire.Decode(payload)
		if err != nil {
			slog.Error("Could not decode the message", "error", err, "payload", string(payload))
			return err
		}
		mp.Messages <- m
	}
}

func (mp *Multiplayer) ForwardRTCMessage(ctx context.Context, msg wire.Message) {
	slog.Debug("Forwarding RTC message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	userId, err := strconv.ParseInt(msg.To, 10, 64)
	if err != nil {
		return
	}

	if user, ok := mp.GetUserSession(userId); ok {
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

type GameRoom struct {
	Ready    bool
	ID       string
	Name     string
	Password string // TODO: Yup, game expects the password in plain-text
	MapID    v1.GameMap

	HostPlayer *UserSession
	CreatedBy  *UserSession

	Players map[int64]*UserSession
}

// ListRooms returns list of all created game rooms.
func (mp *Multiplayer) ListRooms() map[string]*GameRoom {
	mp.roomsMutex.RLock()
	defer mp.roomsMutex.RUnlock()

	return mp.Rooms
}

func (mp *Multiplayer) GetRoom(roomId string) (GameRoom, bool) {
	mp.roomsMutex.RLock()
	defer mp.roomsMutex.RUnlock()

	room, found := mp.Rooms[roomId]
	if !found {
		return GameRoom{}, false
	}
	return *room, found
}

// CreateRoom creates new game room.
func (mp *Multiplayer) CreateRoom(room GameRoom) GameRoom {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	mp.Rooms[room.ID] = &room
	return room
}

// DestroyRoom deletes an existing game room.
func (mp *Multiplayer) DestroyRoom(roomId string) {
	delete(mp.Rooms, roomId)
}

// JoinRoom adds a player to an existing game room.
func (mp *Multiplayer) JoinRoom(roomId string, userId int64, ipAddr string) (GameRoom, error) {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	mp.sessionMutex.Lock()
	defer mp.sessionMutex.Unlock()

	// Finding the user session of the player who joins
	joiningPlayer, found := mp.sessions[userId]
	if !found {
		return GameRoom{}, fmt.Errorf("user session %d not found", userId)
	}

	// Find the game room
	room, found := mp.Rooms[roomId]
	if !found {
		return GameRoom{}, fmt.Errorf("room %s not found", roomId)
	}

	// Check if player was already added to game room
	if _, ok := room.Players[userId]; ok {
		return GameRoom{}, fmt.Errorf("user session %d already joined", userId)
	}

	// Override the IP address
	joiningPlayer.IPAddress = ipAddr
	joiningPlayer.GameID = room.ID

	// Update the game room
	room.Players[userId] = joiningPlayer

	return *room, nil
}

func (mp *Multiplayer) HandleLeaveRoom(ctx context.Context, msg wire.Message) {
	player, ok := msg.Content.(wire.Player)
	if !ok {
		return
	}

	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	mp.sessionMutex.Lock()
	defer mp.sessionMutex.Unlock()

	joinedPlayer, found := mp.sessions[player.UserID]
	if !found {
		return
	}

	room, ok := mp.Rooms[joinedPlayer.GameID]
	if ok {
		return
	}

	delete(room.Players, joinedPlayer.UserID)

	for id, session := range room.Players {
		session.Send(ctx, wire.Compose(wire.LeaveRoom, wire.Message{
			To:   strconv.Itoa(int(id)),
			From: strconv.Itoa(int(joinedPlayer.UserID)),
			Type: wire.LeaveRoom,
			Content: wire.Player{
				UserID:      joinedPlayer.UserID,
				Username:    joinedPlayer.User.Username,
				CharacterID: joinedPlayer.Character.CharacterID,
				ClassType:   joinedPlayer.Character.ClassType,
				IPAddress:   joinedPlayer.IPAddress,
			},
		}))
	}

	if len(room.Players) == 0 {
		mp.DestroyRoom(room.ID)
	}
}

func (mp *Multiplayer) AnnounceJoin(room GameRoom, userId int64) {
	mp.sessionMutex.Lock()

	// Finding the user session of the player who joins
	joinedPlayer, found := mp.sessions[userId]
	if !found {
		mp.sessionMutex.Unlock()
		return
	}
	mp.sessionMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	for id, session := range room.Players {
		if id == userId {
			continue
		}
		if userId == 0 {
			panic("userId is zero")
		}
		session.Send(ctx, wire.Compose(wire.JoinRoom, wire.Message{
			To:   strconv.Itoa(int(id)),
			From: strconv.Itoa(int(userId)),
			Type: wire.JoinRoom,
			Content: wire.Player{
				UserID:      joinedPlayer.UserID,
				Username:    joinedPlayer.User.Username,
				CharacterID: joinedPlayer.Character.CharacterID,
				ClassType:   joinedPlayer.Character.ClassType,
				IPAddress:   joinedPlayer.IPAddress,
			},
		}))
	}
}

// SetRoomReady notifies the LobbyRoom that it can start accepting players.
func (mp *Multiplayer) SetRoomReady(ctx context.Context, msg wire.Message) {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	roomId, ok := msg.Content.(string)
	if !ok {
		return
	}

	lobbyRoom, ok := mp.Rooms[roomId]
	if !ok {
		return
	}

	fromUserId, _ := strconv.ParseInt(msg.From, 10, 64)
	user, ok := mp.GetUserSession(fromUserId)
	if !ok {
		return
	}

	user.GameID = roomId
	lobbyRoom.Ready = true
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
	if et != wire.JoinLobby {
		return fmt.Errorf("inapprioprate event type")
	}

	session.Character.CharacterID = m.Content.CharacterID
	session.Character.ClassType = m.Content.ClassType

	session.Send(ctx, []byte{byte(wire.JoinedLobby)})
	return nil
}

// SetPlayerConnected notifies the user has connected to the lobby.
func (mp *Multiplayer) SetPlayerConnected(session *UserSession) {
	players := mp.listSessions()
	mp.AddUserSession(session.UserID, session)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	// Include in response also the player who has just joined
	players = append(players, session.ToPlayer())

	session.SendMessage(ctx, wire.LobbyUsers, wire.Message{
		Type:    wire.LobbyUsers,
		To:      strconv.FormatInt(session.UserID, 10),
		Content: players,
	})

	// Notify all the users
	mp.BroadcastMessage(ctx, wire.ComposeTyped(wire.JoinLobby, wire.MessageContent[wire.Player]{
		From:    strconv.Itoa(int(session.UserID)),
		Type:    wire.JoinLobby,
		Content: session.ToPlayer(),
	}))
}

// SetPlayerDisconnected notifies the user has left the lobby.
func (mp *Multiplayer) SetPlayerDisconnected(session *UserSession) {
	slog.Info("Closing player connection", "user", session.UserID)

	if err := session.wsConn.CloseNow(); err != nil {
		slog.Debug("Could not close the connection", "user", session.UserID, "error", err)
	}
	mp.DeleteUserSession(session.UserID)
	mp.BroadcastMessage(context.TODO(), wire.Compose(wire.LeaveLobby, wire.Message{
		Type:    wire.LeaveLobby,
		From:    strconv.Itoa(int(session.UserID)),
		Content: session.ToPlayer(),
	}))
}

// BroadcastMessage sends a message to all connected users.
func (mp *Multiplayer) BroadcastMessage(ctx context.Context, payload []byte) {
	// slog.Info("Broadcasting message", "type", wire.EventType(payload[0]).String(), "payload", string(payload[1:]))

	mp.forEachSession(func(session *UserSession) bool {
		session.Send(ctx, payload)
		return true
	})
}

// GetUserSession is a thread-safe method to receive a session by ID.
func (mp *Multiplayer) GetUserSession(id int64) (*UserSession, bool) {
	mp.sessionMutex.RLock()
	member, ok := mp.sessions[id]
	mp.sessionMutex.RUnlock()
	return member, ok
}

// AddUserSession is a thread-safe operation to add a session identified by ID.
func (mp *Multiplayer) AddUserSession(id int64, session *UserSession) {
	if _, exists := mp.GetUserSession(id); exists {
		return
	}
	mp.sessionMutex.Lock()
	mp.sessions[id] = session
	mp.sessionMutex.Unlock()
}

// DeleteUserSession is a thread-safe operation to delete a session by ID.
func (mp *Multiplayer) DeleteUserSession(id int64) {
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
