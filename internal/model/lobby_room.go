package model

import (
	"net"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
)

type LobbyRoom struct {
	HostIPAddress net.IP
	Name          string
	Password      string
	MapID         v1.GameMap
}

func (room *LobbyRoom) ToBytes() []byte {
	ipLength := len(room.HostIPAddress)
	nameLength := len(room.Name)
	passLength := len(room.Password)

	buf := make([]byte, ipLength+nameLength+1+passLength+1)

	copy(buf[0:], room.HostIPAddress[:])             // Host IP Address (4 bytes)
	copy(buf[ipLength:], room.Name)                  // Room name (null terminated string)
	buf[ipLength+nameLength] = 0                     // Null byte
	copy(buf[ipLength+nameLength+1:], room.Password) // Room password (null terminated string)

	return buf
}

type LobbyPlayer struct {
	ClassType v1.ClassType
	IPAddress net.IP
	Name      string
}

func (p *LobbyPlayer) ToBytes() []byte {
	buf := make([]byte, 4+4+len(p.Name)+1)
	buf[0] = byte(p.ClassType)     // Class type (4 bytes)
	copy(buf[4:8], p.IPAddress[:]) // IP Address (4 bytes)
	copy(buf[8:], p.Name)          // Character name (null terminated string)
	return buf
}
