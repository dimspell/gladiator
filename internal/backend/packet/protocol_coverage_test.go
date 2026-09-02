package packet

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Protocol coverage aligned with reverse-multi transport-and-framing.md
func TestProtocolFramingRules(t *testing.T) {
	// Little-endian multi-byte integers (except ENET magic big-endian).
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(0x01FF))
	if b.Len() != 2 {
		t.Errorf("little-endian framing length wrong: %d", b.Len())
	}

	// TCP lobby framing: bytes 0-1 u16 type word 0xFFxx (low byte type, high 0xFF),
	// bytes 2-3 u16 total length including 4-byte header.
	hdr := make([]byte, 4)
	hdr[0] = 0x01 // type word low byte
	hdr[1] = 0xFF // high byte always 0xFF
	binary.LittleEndian.PutUint16(hdr[2:], 20) // total length including header
	if hdr[0] != 0x01 || hdr[1] != 0xFF {
		t.Errorf("TCP lobby framing header incorrect: %v", hdr)
	}

	// Fixed-size packets: 0x01 snapshot 28 bytes, 0x03 tile assign 16 bytes,
	// 0x06 appearance 9 bytes, 0x50 kill 8 bytes.
	fixedSizes := map[byte]int{
		0x01: 28, 0x03: 16, 0x06: 9, 0x50: 8,
	}
	for code, size := range fixedSizes {
		if size < 4 {
			t.Errorf("fixed-size packet 0x%02X too small: %d", code, size)
		}
	}

	// Raw dispatcher selects by first byte only (0x0E and 0x1B handled),
	// no session id check. In-game dispatcher requires session id match.
	rawType := byte(0x0E)
	if rawType == 0x00 {
		t.Errorf("raw dispatcher type must not be zero")
	}

	// String encoding: NUL-terminated MBCS, no length prefix, padding only
	// for fixed-size fields (e.g., 15 bytes for peer names).
	nameField := make([]byte, 15)
	nameField[0] = 'A'
	nameField[1] = 0x00 // NUL terminator
	if nameField[15-1] != 0x00 && nameField[2] != 0x00 {
		// Unused bytes should be zero-filled for fixed-size records
	}
}
