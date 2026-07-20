package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestWriteReadFramed_RoundTrip(t *testing.T) {
	var buf bytes.Buffer
	pkt := &RelayPacket{
		Type:    "tcp",
		RoomID:  "r1",
		FromID:  "a",
		ToID:    "b",
		Payload: []byte("hello world"),
	}
	if err := MarshalFramed(&buf, pkt); err != nil {
		t.Fatalf("MarshalFramed: %v", err)
	}
	got := &RelayPacket{}
	if err := UnmarshalFramed(&buf, got); err != nil {
		t.Fatalf("UnmarshalFramed: %v", err)
	}
	if got.Type != pkt.Type || got.RoomID != pkt.RoomID || got.FromID != pkt.FromID || got.ToID != pkt.ToID {
		t.Fatalf("header mismatch: %+v", got)
	}
	if string(got.Payload) != "hello world" {
		t.Fatalf("payload mismatch: %q", got.Payload)
	}
}

func TestReadFramed_RejectsOversizedFrame(t *testing.T) {
	var buf bytes.Buffer
	// Claim a frame larger than MaxFrameSize; the reader must reject it before
	// allocating, so a hostile length prefix cannot exhaust memory.
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], MaxFrameSize+1)
	if _, err := buf.Write(hdr[:]); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := ReadFramed(&buf); err == nil {
		t.Fatal("expected error for oversized frame, got nil")
	}
}

func TestWriteFramed_RejectsOversizedFrame(t *testing.T) {
	var buf bytes.Buffer
	huge := make([]byte, MaxFrameSize+1)
	if err := WriteFramed(&buf, huge); err == nil {
		t.Fatal("expected error writing oversized frame, got nil")
	}
}

func TestFramed_IsLengthPrefixedNotNewline(t *testing.T) {
	// A payload containing a newline must not be confused with a delimiter:
	// framing is purely length-prefixed, so the round-trip must be exact.
	payload := []byte("line1\nline2\n")
	raw, err := json.Marshal(&RelayPacket{Type: "udp", Payload: payload})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteFramed(&buf, raw); err != nil {
		t.Fatalf("WriteFramed: %v", err)
	}
	got, err := ReadFramed(&buf)
	if err != nil {
		t.Fatalf("ReadFramed: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("framed round-trip corrupted payload with newlines")
	}
}
