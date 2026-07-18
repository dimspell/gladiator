// Package types holds the wire message and framing shared between the relay
// backend proxy and the relay server (console). Keeping it here avoids a
// duplicate RelayPacket definition in both packages and the import cycle that
// would result from either package importing the other.
package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// MaxFrameSize bounds a single relay frame payload to prevent memory
// exhaustion from a malformed or hostile length prefix.
const MaxFrameSize = 1 << 20 // 1 MiB

// RelayPacket is the wire message exchanged between a backend proxy and the
// relay server. It is marshaled to JSON and written with length-prefixed
// framing (see WriteFramed / ReadFramed).
type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", "tcp", "udp", "ping"
	RoomID  string `json:"room"`
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

// WriteFramed writes msg to w using a 4-byte little-endian length prefix
// followed by the raw bytes. This makes each message self-delimiting so the
// reader does not depend on newlines or a fixed buffer size.
func WriteFramed(w io.Writer, msg []byte) error {
	if len(msg) > MaxFrameSize {
		return fmt.Errorf("frame too large: %d > %d", len(msg), MaxFrameSize)
	}
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(msg)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("write frame header: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write frame payload: %w", err)
	}
	return nil
}

// ReadFramed reads a single length-prefixed frame from r. It returns the
// raw payload bytes (the caller is responsible for unmarshaling).
func ReadFramed(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n > MaxFrameSize {
		return nil, fmt.Errorf("frame too large: %d > %d", n, MaxFrameSize)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// MarshalFramed marshals pkt to JSON and writes it framed to w.
func MarshalFramed(w io.Writer, pkt *RelayPacket) error {
	data, err := json.Marshal(pkt)
	if err != nil {
		return fmt.Errorf("marshal packet: %w", err)
	}
	return WriteFramed(w, data)
}

// UnmarshalFramed reads one framed JSON packet from r into pkt.
func UnmarshalFramed(r io.Reader, pkt *RelayPacket) error {
	data, err := ReadFramed(r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, pkt); err != nil {
		return fmt.Errorf("unmarshal packet: %w", err)
	}
	return nil
}
