package packet

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
)

const nullTerminator byte = 0x00

type Reader struct {
	cursor int
	data   []byte
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data}
}

// ReadString reads until the 0x00 character (null-terminator)
func (r *Reader) ReadString() (string, error) {
	pos := bytes.IndexByte(r.data[r.cursor:], nullTerminator)
	if pos == -1 {
		return "", io.EOF
	}

	start, end := r.cursor, pos+r.cursor
	if end > len(r.data)-1 {
		return "", io.EOF
	}

	// Skip the null-terminator
	r.cursor += pos + 1

	return string(r.data[start:end]), nil
}

func (r *Reader) ReadRestBytes() ([]byte, error) {
	return r.data[r.cursor:], nil
}

func (r *Reader) ReadNBytes(n int) ([]byte, error) {
	start, end := r.cursor, n+r.cursor
	if end > len(r.data) {
		return nil, io.EOF
	}

	// Move the cursor by number of read bytes
	r.cursor += n

	return r.data[start:end], nil
}

func (r *Reader) ReadUint8() (uint8, error) {
	b, err := r.ReadNBytes(1)
	log.Println(b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *Reader) ReadUint16() (uint16, error) {
	b, err := r.ReadNBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *Reader) ReadUint32() (uint32, error) {
	b, err := r.ReadNBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *Reader) Close() error {
	r.cursor = 0
	r.data = nil
	return nil
}
