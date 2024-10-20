package p2p

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

func isTCPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/tcp", room)
}

func isUDPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/udp", room)
}
