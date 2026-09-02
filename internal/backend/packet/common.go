package packet

import (
	"encoding/binary"
	"net"

	"github.com/dimspell/gladiator/internal/model"
)

// NewHostSwitch builds the HostMigration payload (0x47FF).
// Payload is 8 bytes: flag at 0 (0=self, 1=peer) and IPv4 at 4.
func NewHostSwitch(external bool, ip net.IP) []byte {
	payload := make([]byte, 8)
	if external {
		copy(payload[0:4], []byte{1, 0, 0, 0})
	} else {
		copy(payload[0:4], []byte{0, 0, 0, 0})
	}
	if ip4 := ip.To4(); ip4 != nil {
		copy(payload[4:], ip4)
	} else {
		copy(payload[4:], net.IPv4zero.To4())
	}
	return payload
}

// NewKickPlayer builds a kick payload for HostMigration.
func NewKickPlayer(ip net.IP) []byte {
	payload := make([]byte, 8)
	if ip4 := ip.To4(); ip4 != nil {
		copy(payload[4:], ip4)
	}
	return payload
}

// NewHostRefresh is 0.0.0.0 — tells client to refresh without migration.
func NewHostRefresh() []byte {
	return NewHostSwitch(false, net.IPv4zero)
}

const (
	opLobbyAppendUser byte = 2
	opLobbyRemoveUser byte = 3

	opChatGlobal byte = 4
	opChatLobby  byte = 5

	opSetChannelName byte = 7

	opLobbySystem byte = 18 // 0x12 — system notice / probe

	opUnknown1 byte = 1 //nolint:unused
)

// AppendCharacterToLobby is sent with ReceiveMessage code.
func AppendCharacterToLobby(userName string, classType model.ClassType, idx uint32) []byte {
	buf := make([]byte, 4+4+4+len(userName)+1)

	buf[0] = opLobbyAppendUser                    // Message type
	buf[4] = byte(classType)                      // Class of character
	binary.LittleEndian.PutUint32(buf[8:12], idx) // Index?
	copy(buf[12:], userName)                      // Character name

	return buf
}

// RemoveCharacterFromLobby is sent with ReceiveMessage code.
func RemoveCharacterFromLobby(userName string) []byte {
	buf := make([]byte, 4+4+4+len(userName)+1)

	buf[0] = opLobbyRemoveUser // Message type
	copy(buf[12:], userName)   // Character name

	return buf
}

// NewGlobalMessage creates a new chat message that will be sent to all users, not just the ones in the lobby.
// NewGlobalMessage is sent with ReceiveMessage code.
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

// NewLobbyMessage is sent with ReceiveMessage code.
func NewLobbyMessage(user, text string) []byte {
	// buf := make([]byte, 4+4+4+len(user)+1+len(text)+1+len(unknown)+1)
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1)

	buf[0] = opChatLobby // Message type
	copy(buf[12:], user)
	copy(buf[12+len(user)+1:], text)
	// copy(buf[12+len(user)+1+len(text)+1:], unknown)

	return buf
}

// SetChannelName is sent with ReceiveMessage code.
func SetChannelName(channelName string) []byte {
	buf := make([]byte, 4+4+4+1+len(channelName)+1)

	buf[0] = opSetChannelName   // Message type
	copy(buf[13:], channelName) // Channel name
	return buf
}

// NewAdminNotice is server -> client system message (type 3).
func NewAdminNotice(user, text string) []byte {
	buf := make([]byte, 4+4+4+len(user)+1+len(text)+1)
	buf[0] = opLobbySystem
	copy(buf[12:], user)
	copy(buf[12+len(user)+1:], text)
	return buf
}

// NewNumericProbe is a silent check with three numbers (no UI).
func NewNumericProbe(s1, s2, s3 string) []byte {
	buf := make([]byte, 4+4+4+len(s1)+1+len(s2)+1+len(s3)+1)
	buf[0] = opLobbySystem
	binary.LittleEndian.PutUint32(buf[8:12], 1) // mode 1
	copy(buf[12:], s1)
	copy(buf[12+len(s1)+1:], s2)
	copy(buf[12+len(s1)+1+len(s2)+1:], s3)
	return buf
}
