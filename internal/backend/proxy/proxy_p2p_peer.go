package proxy

import (
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	UserID string

	Addr *redirect.Addressing
	Mode redirect.Mode

	Connection *webrtc.PeerConnection
}

// Terminate is used to terminate the WebRTC connection.
func (p *Peer) Terminate() {
	if p.Connection != nil {
		slog.Debug("Closing of the WebRTC connection", "peerId", p.UserID)

		if err := p.Connection.Close(); err != nil {
			slog.Error("Failed to close the WebRTC connection", "peerId", p.UserID, "error", err)
			return
		}
	}
}
