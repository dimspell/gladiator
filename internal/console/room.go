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
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/metrics"
	"github.com/dimspell/gladiator/internal/wire"
)

// RoomService is a control plane for the lobby, presence and the matchmaking.
type RoomService struct {
	done context.CancelFunc

	// Presence in a lobby
	sessionMutex sync.RWMutex
	sessions     map[int64]*UserSession

	Messages chan wire.Message

	// Game rooms
	roomsMutex sync.RWMutex
	Rooms      map[string]*GameRoom

	RelayService *RelayService
}

func NewRoomService() *RoomService {
	mp := &RoomService{
		sessions: make(map[int64]*UserSession),
		Rooms:    make(map[string]*GameRoom),
		Messages: make(chan wire.Message),
	}
	return mp
}

func (mp *RoomService) Stop() { mp.done() }

func (mp *RoomService) Reset() {
	mp.forEachSession(func(userSession *UserSession) bool {
		if userSession.WebSocket != nil {
			_ = userSession.WebSocket.CloseNow()
		}
		return true
	})
	clear(mp.sessions)
	close(mp.Messages)
	clear(mp.Rooms)
}

func (mp *RoomService) Run(ctx context.Context) {
	ctx, done := context.WithCancel(ctx)
	mp.done = done
	defer done()

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

// HandleIncomingMessage handles the incoming message pump by dispatching
// commands based on the message type.
func (mp *RoomService) HandleIncomingMessage(ctx context.Context, msg wire.Message) {
	slog.Debug("Received a signal message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)
	start := time.Now()
	metrics.MessagesReceived.WithLabelValues(msg.Type.String()).Inc()

	switch msg.Type {
	case wire.Chat:
		mp.BroadcastMessage(ctx, wire.Compose(wire.Chat, wire.Message{
			From:    msg.From,
			Content: msg.Content,
		}))
	case wire.RTCOffer, wire.RTCAnswer, wire.RTCICECandidate:
		mp.ForwardRTCMessage(ctx, msg)
	case wire.SetRoomReady:
		mp.SetRoomReady(msg)
	default:
		// Do nothing but log the event type
		slog.Error("Unhandled event type", "type", msg.Type.String())
		metrics.MultiplayerErrors.WithLabelValues("unhandled_event").Inc()
		metrics.UnhandledMessageTypes.WithLabelValues(msg.Type.String()).Inc()
	}
	metrics.MessageProcessingLatency.Observe(time.Since(start).Seconds())
}

func (mp *RoomService) HandleSession(ctx context.Context, session *UserSession) error {
	startSession := time.Now()
	// Expect the "hello" and send back "welcome" message.
	if err := mp.HandleHello(ctx, session); err != nil {
		metrics.MultiplayerErrors.WithLabelValues("hello").Inc()
		return err
	}

	// Expect the character info, then join and synchronise the state.
	if err := mp.HandleJoinLobby(ctx, session); err != nil {
		metrics.MultiplayerErrors.WithLabelValues("join_lobby").Inc()
		return err
	}

	// Add user to the list of connected players.
	mp.SetPlayerConnected(session)
	metrics.ActiveSessions.Inc()
	metrics.TotalSessions.Inc()

	// Remove the player
	defer func() {
		mp.SetPlayerDisconnected(session)
		metrics.ActiveSessions.Dec()
		sessionDuration := time.Since(startSession).Seconds()
		metrics.PlayerSessionDuration.Observe(sessionDuration)
	}()

	// Handle all the incoming messages.
	for {
		// Register that the user is still being active.
		session.ConnectedAt = time.Now().In(time.UTC)

		payload, err := session.ReadNext(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				metrics.MultiplayerErrors.WithLabelValues("context_canceled").Inc()
				metrics.WebSocketDisconnects.WithLabelValues("context_canceled").Inc()
				return err
			}

			switch state := websocket.CloseStatus(err); state {
			case -1:
				// connection reset by peer
				metrics.WebSocketDisconnects.WithLabelValues("reset_by_peer").Inc()
				return nil
			case websocket.StatusNormalClosure:
				slog.Debug("Closing because of", logging.Error(err))
				metrics.WebSocketDisconnects.WithLabelValues("normal_closure").Inc()
				return err
			default:
				slog.Error("Could not handle the message", logging.Error(err))
				metrics.MultiplayerErrors.WithLabelValues("read_next").Inc()
				metrics.WebSocketDisconnects.WithLabelValues("other_error").Inc()
				return err
			}
		}

		// Enqueue message
		_, m, err := wire.Decode(payload)
		if err != nil {
			slog.Error("Could not decode the message", logging.Error(err), "payload", string(payload))
			metrics.MultiplayerErrors.WithLabelValues("decode").Inc()
			metrics.InvalidPayloads.Inc()
			return err
		}
		metrics.MessagesReceived.WithLabelValues(m.Type.String()).Inc()
		mp.Messages <- m
	}
}

func (mp *RoomService) ForwardRTCMessage(ctx context.Context, msg wire.Message) {
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
func (mp *RoomService) DebugState() {
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

	CreatedAt time.Time // For room lifetime metrics
}

// ListRooms returns list of all created game rooms.
func (mp *RoomService) ListRooms() map[string]*GameRoom {
	mp.roomsMutex.RLock()
	defer mp.roomsMutex.RUnlock()

	return mp.Rooms
}

func (mp *RoomService) GetRoom(roomId string) (GameRoom, bool) {
	mp.roomsMutex.RLock()
	defer mp.roomsMutex.RUnlock()

	room, found := mp.Rooms[roomId]
	if !found {
		return GameRoom{}, false
	}
	return *room, found
}

// CreateRoom creates new game room.
func (mp *RoomService) CreateRoom(hostUserID int64, gameID string, password string, mapID v1.GameMap, hostIpAddress string) (*GameRoom, error) {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	hostSession, found := mp.GetUserSession(hostUserID)
	if !found {
		metrics.MultiplayerErrors.WithLabelValues("create_room_no_user").Inc()
		return nil, fmt.Errorf("user session not found %q", hostUserID)
	}

	if _, exist := mp.Rooms[gameID]; exist {
		metrics.MultiplayerErrors.WithLabelValues("create_room_exists").Inc()
		return nil, fmt.Errorf("room already exists")
	}

	// TODO: Be more gentle with interfacing with the Relay Server
	// if mp.Relay != nil {
	// 	mp.Relay.Server.leaveRoom(fmt.Sprintf("%d", hostUserID), gameID)
	// }

	hostSession.GameID = gameID
	hostSession.IPAddress = hostIpAddress

	room := &GameRoom{
		Ready:      false,
		ID:         gameID,
		Name:       gameID,
		Password:   password,
		MapID:      mapID,
		HostPlayer: hostSession,
		CreatedBy:  hostSession,
		Players:    map[int64]*UserSession{hostSession.UserID: hostSession},
		CreatedAt:  time.Now().In(time.UTC),
	}
	mp.Rooms[gameID] = room
	metrics.MultiplayerActiveRooms.Inc()
	metrics.MultiplayerTotalRoomsCreated.Inc()
	metrics.PlayersPerRoom.WithLabelValues(gameID).Set(float64(len(room.Players)))
	return room, nil
}

// DestroyRoom deletes an existing game room.
func (mp *RoomService) DestroyRoom(roomId string) {
	room, ok := mp.Rooms[roomId]
	if ok {
		lifetime := time.Since(room.CreatedAt).Seconds()
		metrics.RoomLifetime.Observe(lifetime)
		metrics.PlayersPerRoom.DeleteLabelValues(roomId)
	}
	delete(mp.Rooms, roomId)
	metrics.MultiplayerActiveRooms.Dec()
}

// JoinRoom adds a player to an existing game room.
func (mp *RoomService) JoinRoom(roomId string, userId int64, ipAddr string) (GameRoom, error) {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	mp.sessionMutex.Lock()
	defer mp.sessionMutex.Unlock()

	// Finding the user session of the player who joins
	joiningPlayer, found := mp.sessions[userId]
	if !found {
		metrics.MultiplayerErrors.WithLabelValues("join_room_no_user").Inc()
		return GameRoom{}, fmt.Errorf("user session %d not found", userId)
	}

	// Find the game room
	room, found := mp.Rooms[roomId]
	if !found {
		metrics.MultiplayerErrors.WithLabelValues("join_room_no_room").Inc()
		return GameRoom{}, fmt.Errorf("room %s not found", roomId)
	}

	// Check if player was already added to the game room
	if _, ok := room.Players[userId]; ok {
		slog.Warn("User already joined a room", "room", roomId, "user", userId)
		metrics.MultiplayerErrors.WithLabelValues("join_room_already_joined").Inc()
		return GameRoom{}, fmt.Errorf("user session %d already joined", userId)
	}

	// Override the IP address
	joiningPlayer.IPAddress = ipAddr
	joiningPlayer.GameID = room.ID
	joiningPlayer.JoinedAt = time.Now().In(time.UTC)

	// Update the game room
	room.Players[userId] = joiningPlayer
	metrics.RoomJoins.Inc()
	metrics.PlayersPerRoom.WithLabelValues(roomId).Set(float64(len(room.Players)))
	return *room, nil
}

// LeaveRoom removes a player from a game room.
func (mp *RoomService) LeaveRoom(ctx context.Context, session *UserSession) {
	mp.roomsMutex.Lock()
	defer mp.roomsMutex.Unlock()

	room, ok := mp.Rooms[session.GameID]
	if !ok {
		return
	}

	// Was the player the game host?
	playerWasHost := room.HostPlayer.UserID == session.UserID

	delete(room.Players, session.UserID)
	metrics.RoomLeaves.Inc()
	metrics.PlayersPerRoom.WithLabelValues(room.ID).Set(float64(len(room.Players)))

	if len(room.Players) == 0 {
		// There is nobody in the room, so we can destroy it
		mp.DestroyRoom(room.ID)
		return
	}

	if playerWasHost {
		// Find the user who will become the new host
		room.HostPlayer = mp.GetNextHost(room)
		metrics.HostMigrations.Inc()
	}

	for id, player := range room.Players {
		player.Send(ctx, wire.Compose(wire.LeaveRoom, wire.Message{
			To:   strconv.Itoa(int(id)),
			From: strconv.Itoa(int(session.UserID)),
			Type: wire.LeaveRoom,
			Content: wire.Player{
				UserID:      session.UserID,
				Username:    session.User.Username,
				CharacterID: session.Character.CharacterID,
				ClassType:   session.Character.ClassType,
				IPAddress:   session.IPAddress,
			},
		}))

		if playerWasHost && room.HostPlayer != nil {
			player.Send(ctx, wire.Compose(wire.HostMigration, wire.Message{
				To:   strconv.Itoa(int(id)),
				From: strconv.Itoa(int(room.HostPlayer.UserID)),
				Type: wire.HostMigration,
				Content: wire.Player{
					UserID:      room.HostPlayer.UserID,
					Username:    room.HostPlayer.User.Username,
					CharacterID: room.HostPlayer.Character.CharacterID,
					ClassType:   room.HostPlayer.Character.ClassType,
					IPAddress:   room.HostPlayer.IPAddress,
				},
			}))
		}
	}

	// mp.Relay.Server.switchHost(roomID, peerID)
}

// GetNextHost returns the next host of the game room.
func (mp *RoomService) GetNextHost(room *GameRoom) *UserSession {
	var earliest *UserSession

	// Find the player who joined the room earliest
	for _, player := range room.Players {
		if earliest == nil || player.JoinedAt.Before(earliest.JoinedAt) {
			earliest = player
		}
	}

	return earliest
}

func (mp *RoomService) AnnounceJoin(room GameRoom, userId int64) {
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
func (mp *RoomService) SetRoomReady(msg wire.Message) {
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

	lobbyRoom.Ready = true
	metrics.RoomReadyEvents.Inc()
}

func (mp *RoomService) HandleHello(ctx context.Context, session *UserSession) error {
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

func (mp *RoomService) HandleJoinLobby(ctx context.Context, session *UserSession) error {
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
func (mp *RoomService) SetPlayerConnected(session *UserSession) {
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
func (mp *RoomService) SetPlayerDisconnected(session *UserSession) {
	slog.Info("Closing player connection", "user", session.UserID)

	// Close the websocket connection
	if err := session.WebSocket.CloseNow(); err != nil {
		slog.Debug("Could not close the connection", "user", session.UserID, logging.Error(err))
	}

	// Kick the user from the game room (if any)
	mp.LeaveRoom(context.Background(), session)

	// Notify the relay server the user has disconnected
	if mp.RelayService != nil {
		slog.Info("Closing relay connection", "user", session.UserID)
		mp.RelayService.Server.leaveRoom(fmt.Sprintf("%d", session.UserID), session.GameID)
	}

	// Delete the session from the map
	mp.DeleteUserSession(session.UserID)

	// Notify all the users
	mp.BroadcastMessage(context.Background(), wire.Compose(wire.LeaveLobby, wire.Message{
		Type:    wire.LeaveLobby,
		From:    strconv.Itoa(int(session.UserID)),
		Content: session.ToPlayer(),
	}))
}

// BroadcastMessage sends a message to all connected users.
func (mp *RoomService) BroadcastMessage(ctx context.Context, payload []byte) {
	// slog.Info("Broadcasting message", "type", wire.EventType(payload[0]).String(), "payload", string(payload[1:]))
	metrics.MessagesBroadcasted.Inc()
	mp.forEachSession(func(session *UserSession) bool {
		session.Send(ctx, payload)
		return true
	})
}

// GetUserSession is a thread-safe method to receive a session by ID.
func (mp *RoomService) GetUserSession(id int64) (*UserSession, bool) {
	mp.sessionMutex.RLock()
	member, ok := mp.sessions[id]
	mp.sessionMutex.RUnlock()
	return member, ok
}

// AddUserSession is a thread-safe operation to add a session identified by ID.
func (mp *RoomService) AddUserSession(id int64, session *UserSession) {
	if _, exists := mp.GetUserSession(id); exists {
		return
	}
	mp.sessionMutex.Lock()
	mp.sessions[id] = session
	mp.sessionMutex.Unlock()
}

// DeleteUserSession is a thread-safe operation to delete a session by ID.
func (mp *RoomService) DeleteUserSession(id int64) {
	mp.sessionMutex.Lock()
	delete(mp.sessions, id)
	mp.sessionMutex.Unlock()
}

// forEachSession is a thread-safe method to iterate over all session entries.
func (mp *RoomService) forEachSession(f func(session *UserSession) bool) {
	mp.sessionMutex.RLock()
	defer mp.sessionMutex.RUnlock()
	for _, member := range mp.sessions {
		if next := f(member); !next {
			return
		}
	}
}

// listSession is a thread-safe method to retrieve the session list.
func (mp *RoomService) listSessions() []wire.Player {
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

// In Multiplayer, add a method to register relay event hooks
func (mp *RoomService) RegisterRelayHooks(relay *RelayServer) {
	relay.OnJoin = func(eventType, peerID, roomID string) {
		mp.HandleRelayJoin(eventType, peerID, roomID)
	}
	relay.OnLeave = func(eventType, peerID, roomID string) {
		mp.HandleRelayLeave(eventType, peerID, roomID)
	}
	relay.OnDelete = func(eventType, peerID, roomID string) {
		mp.HandleRelayDelete(eventType, peerID, roomID)
	}
}

// Stub handler methods (implement as needed)
func (mp *RoomService) HandleRelayJoin(eventType, peerID, roomID string) {
	// TODO: Implement join event handling
}

func (mp *RoomService) HandleRelayLeave(eventType, peerID, roomID string) {
	userID, err := strconv.ParseInt(peerID, 10, 64)
	if err != nil {
		return
	}
	sess, found := mp.GetUserSession(userID)
	if !found {
		return
	}
	mp.LeaveRoom(context.Background(), sess)
}

func (mp *RoomService) HandleRelayDelete(eventType, peerID, roomID string) {
	// TODO: Implement delete event handling
}
