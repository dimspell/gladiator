package packet_test

import (
	"net"
	"testing"

	"github.com/dimspell/gladiator/internal/backend/packet"
)

func TestNewHostSwitch(t *testing.T) {
	testCases := []struct {
		name     string
		external bool
		ip       net.IP
		want     []byte
	}{
		{
			name:     "internal host switch",
			external: false,
			ip:       net.IPv4(127, 0, 0, 1),
			want:     []byte{0, 0, 0, 0, 127, 0, 0, 1},
		},
		{
			name:     "external host switch",
			external: true,
			ip:       net.IPv4(192, 168, 1, 1),
			want:     []byte{1, 0, 0, 0, 192, 168, 1, 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := packet.NewHostSwitch(tc.external, tc.ip)

			if len(got) != len(tc.want) {
				t.Errorf("NewHostSwitch() returned wrong length payload: got %d, want %d", len(got), len(tc.want))
			}

			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("NewHostSwitch() byte at position %d: got %d, want %d", i, got[i], tc.want[i])
				}
			}
		})
	}
}
