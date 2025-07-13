package packet

import (
	"encoding/binary"
	"log/slog"
)

func Split(buf []byte) [][]byte {
	if len(buf) < 4 {
		return [][]byte{buf}
	}

	var packets [][]byte
	var offset int
	for i := 0; i < 10; i++ {
		if (offset + 4) > len(buf) { // 0 + 4 > 28
			break
		}

		// The opcode in the header must start with 255
		header := buf[offset]
		if header != 255 {
			break
		}

		length := int(binary.LittleEndian.Uint16(buf[offset+2 : offset+4]))

		// Ignore oversize packets
		if length > len(buf)+offset {
			slog.Error("Oversize packet", "data", buf[offset:])
			break
		}

		packets = append(packets, buf[offset:offset+length])
		offset += length
	}
	return packets
}
