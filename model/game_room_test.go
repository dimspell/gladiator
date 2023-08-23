package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameRoom_ToBytes(t *testing.T) {
	room := LobbyRoom{
		HostIPAddress: [4]byte{127, 0, 0, 1},
		Name:          "Game room",
		Password:      "secret123",
	}

	assert.Equal(t, []byte{
		127, 0, 0, 1,
		71, 97, 109, 101, 32, 114, 111, 111, 109, 0,
		115, 101, 99, 114, 101, 116, 49, 50, 51, 0,
	}, room.ToBytes())
}

func BenchmarkGameRoom_ToBytes(b *testing.B) {
	b.StopTimer()
	room := LobbyRoom{
		HostIPAddress: [4]byte{127, 0, 0, 1},
		Name:          "Game room",
		Password:      "secret123",
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		room.ToBytes()
	}
}
