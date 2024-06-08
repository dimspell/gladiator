package backend

import (
	"bytes"
	"fmt"
)

const nullTerminator byte = 0x00

type PacketReader struct {
	cursor int
	data   []byte
}

func NewPacketReader(data []byte) *PacketReader {
	return &PacketReader{data: data}
}

// ReadString reads until the 0x00 character (null-terminator)
func (r *PacketReader) ReadString() (string, error) {
	pos := bytes.IndexByte(r.data[r.cursor:], nullTerminator)
	if pos == -1 {
		return "", fmt.Errorf("no string found")
	}

	start, end := r.cursor, pos+r.cursor
	if end > len(r.data)-1 {
		return "", fmt.Errorf("out of bounds")
	}

	// Move the cursor by the position of the null-terminator and add 1 to skip
	// the delimiting character.
	r.cursor += pos + 1

	return string(r.data[start:end]), nil
}

func (r *PacketReader) RestBytes() ([]byte, error) {
	return r.data[r.cursor:], nil
}
