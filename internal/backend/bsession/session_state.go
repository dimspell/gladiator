package bsession

import (
	"slices"
	"sync"

	"github.com/dimspell/gladiator/internal/wire"
)

type SessionState struct {
	sync.RWMutex

	// lobbyUsers contains list of players who are connected to lobby server.
	lobbyUsers []wire.Player
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
