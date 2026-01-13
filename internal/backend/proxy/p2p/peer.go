package p2p

import (
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/pion/webrtc/v4"
)

// Peer represents a connected player via WebRTC.
type Peer struct {
	peerID      string
	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	connected   bool
	logger      *slog.Logger
}

// Close terminates the peer connection.
func (p *Peer) Close() {
	if p.dataChannel != nil {
		if err := p.dataChannel.Close(); err != nil {
			p.logger.Debug("Failed to close data channel", logging.Error(err))
		}
	}
	if p.connection != nil {
		if err := p.connection.Close(); err != nil {
			p.logger.Debug("Failed to close peer connection", logging.Error(err))
		}
	}
}
