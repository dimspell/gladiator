package model

import (
	"encoding/binary"
)

type GameRoom struct {
	Lobby   LobbyRoom
	MapID   uint32
	Players []LobbyPlayer
}

type LobbyRoom struct {
	HostIPAddress [4]byte
	Name          string
	Password      string
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

func (r *GameRoom) Details() []byte {
	buf := []byte{}
	buf = binary.LittleEndian.AppendUint32(buf, r.MapID)
	for _, player := range r.Players {
		buf = append(buf, player.ToBytes()...)
	}
	return buf
}

type LobbyPlayer struct {
	ClassType ClassType
	IPAddress [4]byte
	Name      string
}

func (p *LobbyPlayer) ToBytes() []byte {
	buf := make([]byte, 4+4+len(p.Name)+1)
	buf[0] = byte(p.ClassType)     // Class type (4 bytes)
	copy(buf[4:8], p.IPAddress[:]) // IP Address (4 bytes)
	copy(buf[8:], p.Name)          // Character name (null terminated string)
	return buf
}
