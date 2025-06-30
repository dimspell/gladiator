package backend

import (
	"encoding/binary"
	"github.com/dimspell/gladiator/internal/model"
)

const (
	opLobbyAppendUser byte = 2
	opLobbyRemoveUser byte = 3

	opChatGlobal byte = 4
	opChatLobby  byte = 5

	opSetChannelName byte = 7

	opUnknown1  byte = 1
	opUnknown17 byte = 18 // 0x11? 0x12?
)

func AppendCharacterToLobby(userName string, classType model.ClassType, idx uint32) []byte {
	buf := make([]byte, 4+4+4+len(userName)+1)

	buf[0] = opLobbyAppendUser                    // Message type
	buf[4] = byte(classType)                      // Class of character
	binary.LittleEndian.PutUint32(buf[8:12], idx) // Index?
	copy(buf[12:], userName)                      // Character name

	return buf
}

func RemoveCharacterFromLobby(userName string) []byte {
	buf := make([]byte, 4+4+4+len(userName)+1)

	buf[0] = opLobbyRemoveUser // Message type
	copy(buf[12:], userName)   // Character name

	return buf
}

// NewGlobalMessage creates a new chat message that will be sent to all users, not just the ones in the lobby.
func NewGlobalMessage(user, text string) []byte {
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1)

	buf[0] = opChatGlobal            // Message type
	copy(buf[12:], user)             // User name
	copy(buf[12+len(user)+1:], text) // Text of message

	return buf
}

// Note: These are very similar - prints a message using a red text, ignoring the username
// session.Send(packet.ReceiveMessage, NewLobbyMessage("admin", "admin lobby test", "")) - this will be displayed in lobby only
// session.Send(packet.ReceiveMessage, NewGlobalMessage("admin", "admin global test")) - this will be displayed in-game also

func NewLobbyMessage(user, text string) []byte {
	//buf := make([]byte, 4+4+4+len(user)+1+len(text)+1+len(unknown)+1)
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1)

	buf[0] = opChatLobby // Message type
	copy(buf[12:], user)
	copy(buf[12+len(user)+1:], text)
	//copy(buf[12+len(user)+1+len(text)+1:], unknown)

	return buf
}

func SetChannelName(channelName string) []byte {
	buf := make([]byte, 4+4+4+1+len(channelName)+1)

	buf[0] = opSetChannelName   // Message type
	copy(buf[13:], channelName) // Channel name
	return buf
}

// 18?
// resp := []byte{255, opReceiveMessage, 0, 0}
// resp = append(resp, 18, 0, 0, 0)
// resp = append(resp, 0, 0, 0, 0)
// resp = append(resp, 1, 0, 0, 0)
// resp = append(resp, nullTerminatedString("100")...)
// resp = append(resp, nullTerminatedString("200")...)
// resp = append(resp, nullTerminatedString("300")...)
// binary.LittleEndian.PutUint16(resp[2:4], uint16(len(resp)))
// conn.Write(resp)
