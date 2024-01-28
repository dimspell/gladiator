package backend

import (
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

func AppendCharacterToLobby(characterName string, classType model.ClassType, idx uint32) []byte {
	buf := make([]byte, 4+4+4+len(characterName)+1)

	buf[0] = 2                                    // Message type
	buf[4] = byte(classType)                      // Class of character
	binary.LittleEndian.PutUint32(buf[8:12], idx) // Unknown 28?
	copy(buf[12:], characterName)                 // Character name

	return buf
}

func RemoveCharacterFromLobby(characterName string) []byte {
	buf := make([]byte, 4+4+4+len(characterName)+1)

	buf[0] = 3                    // Message type
	copy(buf[12:], characterName) // Character name

	return buf
}

func NewLobbyMessage(user, text string) []byte {
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1)

	buf[0] = 4                       // Message type
	copy(buf[12:], user)             // User name
	copy(buf[12+len(user)+1:], text) // Text of message

	return buf
}

func NewSystemMessage(user, text, unknown string) []byte {
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1+len(unknown)+1)

	buf[0] = 5 // Message type
	copy(buf[12:], user)
	copy(buf[12+len(user)+1:], text)
	copy(buf[12+len(user)+1+len(text)+1:], unknown)

	return buf
}

func SetChannelName(channelName string) {
	buf := make([]byte, 4+4+4+1+len(channelName)+1)

	buf[0] = 3                  // Message type
	copy(buf[13:], channelName) // Character name
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
