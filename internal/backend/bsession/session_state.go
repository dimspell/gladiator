package bsession

import (
	"fmt"
	"slices"
	"sync"

	"github.com/dimspell/gladiator/internal/wire"
)

type SessionState struct {
	sync.RWMutex

	gameRoom *GameRoom

	// lobbyUsers contains list of players who are connected to lobby server.
	lobbyUsers []wire.Player
}

func (s *SessionState) GameRoom() (*GameRoom, error) {
	s.RLock()
	defer s.RUnlock()
	if s.gameRoom == nil {
		return nil, fmt.Errorf("game room not set in session state")
	}
	return s.gameRoom, nil
}

func (s *SessionState) SetGameRoom(gameRoom *GameRoom) {
	s.Lock()
	s.gameRoom = gameRoom
	s.Unlock()
}

func (s *SessionState) UpdateLobbyUsers(users []wire.Player) {
	s.Lock()
	s.lobbyUsers = users
	s.Unlock()
}

func (s *SessionState) GetLobbyUsers() []wire.Player {
	s.RLock()
	defer s.RUnlock()
	return s.lobbyUsers
}

func (s *SessionState) DeleteLobbyUser(userIdToDelete int64) {
	s.Lock()
	s.lobbyUsers = slices.DeleteFunc(s.lobbyUsers, func(player wire.Player) bool {
		return userIdToDelete == player.UserID
	})
	s.Unlock()
}
