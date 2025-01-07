package packet

import (
	"encoding/binary"
)

func EncodePacket(packetType PacketType, data []byte) []byte {
	length := len(data) + 4
	payload := make([]byte, length)

	// Header
	payload[0] = 255
	payload[1] = byte(packetType)
	binary.LittleEndian.PutUint16(payload[2:4], uint16(length))

	// Data
	copy(payload[4:], data)

	return payload
}
