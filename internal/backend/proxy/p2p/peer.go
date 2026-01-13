package p2p

import (
	"log/slog"
	"sync"

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

	mu            sync.Mutex
	outboundQueue [][]byte
}

// Send sends a payload over the data channel, or queues it until the channel is open.
// Queueing is important because the local game client may start sending packets
// before WebRTC is fully negotiated.
func (p *Peer) Send(payload []byte) error {
	p.mu.Lock()
	dc := p.dataChannel
	if dc == nil || dc.ReadyState() != webrtc.DataChannelStateOpen {
		// Keep the queue bounded to avoid unbounded memory growth.
		const maxQueued = 256
		if len(p.outboundQueue) < maxQueued {
			p.outboundQueue = append(p.outboundQueue, append([]byte(nil), payload...))
		} else {
			p.logger.Warn("Dropping outbound p2p packet; queue full", "peerID", p.peerID, "len", len(payload))
		}
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	return dc.Send(payload)
}

func (p *Peer) setDataChannel(dc *webrtc.DataChannel) {
	p.mu.Lock()
	p.dataChannel = dc
	p.mu.Unlock()

	dc.OnOpen(func() {
		p.mu.Lock()
		queued := p.outboundQueue
		p.outboundQueue = nil
		p.mu.Unlock()

		for _, payload := range queued {
			if err := dc.Send(payload); err != nil {
				p.logger.Warn("Failed flushing queued payload", logging.Error(err))
				return
			}
		}
	})
}

// Close terminates the peer connection.
func (p *Peer) Close() {
	p.mu.Lock()
	dc := p.dataChannel
	pc := p.connection
	p.dataChannel = nil
	p.connection = nil
	p.outboundQueue = nil
	p.mu.Unlock()

	if dc != nil {
		if err := dc.Close(); err != nil {
			p.logger.Debug("Failed to close data channel", logging.Error(err))
		}
	}
	if pc != nil {
		if err := pc.Close(); err != nil {
			p.logger.Debug("Failed to close peer connection", logging.Error(err))
		}
	}
}
