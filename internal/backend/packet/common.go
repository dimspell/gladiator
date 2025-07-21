package packet

import "net"

func NewHostSwitch(external bool, ip net.IP) []byte {
	payload := make([]byte, 8)

	if external {
		copy(payload[0:4], []byte{1, 0, 0, 0})
	} else {
		copy(payload[0:4], []byte{0, 0, 0, 0})
	}

	copy(payload[4:], ip.To4())

	return payload
}

func NewKickPlayer(ip net.IP) []byte {
	payload := make([]byte, 8)
	copy(payload[0:4], []byte{0, 0, 0, 0})
	copy(payload[4:], ip.To4())

	return payload
}
