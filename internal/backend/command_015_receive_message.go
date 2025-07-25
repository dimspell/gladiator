package backend

import (
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

// Deprecated: Use packet.AppendCharacterToLobby.
func AppendCharacterToLobby(userName string, classType model.ClassType, idx uint32) []byte {
	return packet.AppendCharacterToLobby(userName, classType, idx)
}

// Deprecated: Use packet.RemoveCharacterFromLobby.
func RemoveCharacterFromLobby(userName string) []byte {
	return packet.RemoveCharacterFromLobby(userName)
}

// Deprecated: Use packet.NewGlobalMessage.
func NewGlobalMessage(user, text string) []byte {
	return packet.NewGlobalMessage(user, text)
}

// Deprecated: Use packet.NewLobbyMessage.
func NewLobbyMessage(user, text string) []byte {
	return packet.NewLobbyMessage(user, text)
}

// Deprecated: Use packet.SetChannelName.
func SetChannelName(channelName string) []byte {
	return packet.SetChannelName(channelName)
}
