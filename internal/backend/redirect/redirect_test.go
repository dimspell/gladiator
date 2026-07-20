package redirect

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTCPRedirect_OtherUserHasJoined_ReturnsTCPListener(t *testing.T) {
	addr := &Addressing{
		IP:      net.IPv4(127, 0, 0, 1),
		TCPPort: "0",
		UDPPort: "0",
	}
	r, err := NewTCPRedirect(OtherUserHasJoined, addr)
	require.NoError(t, err)
	require.IsType(t, &ListenerTCP{}, r, "OtherUserHasJoined must return a TCP listener, not a Noop or UDP listener")
	_ = r.Close()
}

func TestNewUDPRedirect_OtherUserHasJoined_ReturnsUDPListener(t *testing.T) {
	addr := &Addressing{
		IP:      net.IPv4(127, 0, 0, 1),
		TCPPort: "0",
		UDPPort: "0",
	}
	r, err := NewUDPRedirect(OtherUserHasJoined, addr)
	require.NoError(t, err)
	require.IsType(t, &ListenerUDP{}, r, "OtherUserHasJoined must return a UDP listener")
	_ = r.Close()
}

func TestNewTCPRedirect_OtherUserIsHost_ReturnsTCPListener(t *testing.T) {
	addr := &Addressing{
		IP:      net.IPv4(127, 0, 0, 1),
		TCPPort: "0",
		UDPPort: "0",
	}
	r, err := NewTCPRedirect(OtherUserIsHost, addr)
	require.NoError(t, err)
	require.IsType(t, &ListenerTCP{}, r)
	_ = r.Close()
}
