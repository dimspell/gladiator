package packet

import "encoding/binary"

func Split(buf []byte) [][]byte {
	if len(buf) < 4 {
		return [][]byte{buf}
	}

	var packets [][]byte
	var offset int
	for i := 0; i < 10; i++ {
		if (offset + 4) > len(buf) {
			break
		}

		length := int(binary.LittleEndian.Uint16(buf[offset+2 : offset+4]))
		packets = append(packets, buf[offset:offset+length])
		offset += length
	}
	return packets
}
