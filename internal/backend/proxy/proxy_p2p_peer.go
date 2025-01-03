package proxy

import (
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	PeerUserID string

	Addr *redirect.Addressing
	Mode redirect.Mode

	Connection *webrtc.PeerConnection
}

// Close is used to terminate the WebRTC connection.
func (p *Peer) Close() {
	if p.Connection != nil {
		slog.Debug("Closing of the WebRTC connection", "peerId", p.PeerUserID)

		if err := p.Connection.Close(); err != nil {
			slog.Error("Failed to close the WebRTC connection", "peerId", p.PeerUserID, "error", err)
			return
		}
	}
}
