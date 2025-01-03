package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	UserID string

	Addr *redirect.Addressing
	Mode redirect.Mode

	Connection *webrtc.PeerConnection
}

func (p *Peer) setupPeerConnection(session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
	p.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", player.UserID, "state", connectionState.String())
	})

	p.Connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		if err := session.SendRTCICECandidate(context.TODO(), candidate.ToJSON(), player.ID()); err != nil {
			slog.Error("Could not send ICE candidate", "from", session.GetUserID(), "to", player.UserID, "error", err)
		}
	})

	p.Connection.OnNegotiationNeeded(func() {
		if err := p.handleNegotiation(session, player, sendRTCOffer); err != nil {
			slog.Error("failed to handle negotiation", "error", err)
		}
	})

	return nil
}

func (p *Peer) handleNegotiation(session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	if sendRTCOffer {
		if err := session.SendRTCOffer(context.TODO(), offer, player.ID()); err != nil {
			return fmt.Errorf("failed to send RTC offer: %w", err)
		}
	}

	return nil
}

func (p *Peer) createDataChannels(newRedirect redirect.NewRedirect) error {
	if guestTCP, guestUDP, err := newRedirect(p.Mode, p.Addr); err == nil {
		if guestTCP != nil {
			if err := p.createDataChannel("tcp", guestTCP); err != nil {
				return err
			}
		}
		if guestUDP != nil {
			if err := p.createDataChannel("udp", guestUDP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Peer) createDataChannel(label string, redir redirect.Redirect) error {
	dc, err := p.Connection.CreateDataChannel(label, nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", label, err)
	}

	pipe := NewPipe(dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel", "label", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("dataChannel has closed", "label", label)

		pipe.Close()
	})

	return nil
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
