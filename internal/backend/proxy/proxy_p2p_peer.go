package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	UserID     string
	Addr       *redirect.Addressing
	Mode       redirect.Mode
	Connection *webrtc.PeerConnection

	// mu sync.RWMutex
}

func NewPeer(r *IpRing, userId string, isCurrentUser, isHost bool) *Peer {
	switch true {
	case isCurrentUser:
		return &Peer{
			UserID: userId,
			Addr:   &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:   redirect.CurrentUserIsHost,
		}
	case !isCurrentUser && isHost:
		ip, portTCP, portUDP, err := r.NextAddr()
		if err != nil {
			slog.Error("Failed to get the next address", "error", err)
			panic(err)
			return nil
		}
		return &Peer{
			UserID: userId,
			Addr:   &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP},
			Mode:   redirect.OtherUserIsHost,
		}
	case !isCurrentUser && !isHost:
		ip, _, portUDP, err := r.NextAddr()
		if err != nil {
			slog.Error("Failed to get the next address", "error", err)
			panic(err)
			return nil
		}
		return &Peer{
			UserID: userId,
			Addr:   &redirect.Addressing{IP: ip, TCPPort: "", UDPPort: portUDP},
			Mode:   redirect.OtherUserHasJoined,
		}
	default:
		return &Peer{
			UserID: userId,
			Addr:   &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:   redirect.OtherUserIsJoining,
		}
	}
}

func (p *Peer) setupPeerConnection(ctx context.Context, session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
	p.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed",
			"userId", player.UserID,
			"state", connectionState.String())
	})

	p.Connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		if err := session.SendRTCICECandidate(ctx, candidate.ToJSON(), player.ID()); err != nil {
			slog.Error("Could not send ICE candidate",
				"fromID", session.GetUserID(),
				"toID", player.UserID,
				"error", err)
		}
	})

	p.Connection.OnNegotiationNeeded(func() {
		if err := p.handleNegotiation(ctx, session, player, sendRTCOffer); err != nil {
			slog.Error("Failed to handle negotiation",
				"userId", player.UserID,
				"error", err)
		}
	})

	return nil
}

func (p *Peer) handleNegotiation(ctx context.Context, session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer for peer %s: %w", player.UserID, err)
	}

	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description for peer %s: %w", player.UserID, err)
	}

	if sendRTCOffer {
		if err := session.SendRTCOffer(ctx, offer, player.ID()); err != nil {
			return fmt.Errorf("failed to send RTC offer to peer %s: %w", player.UserID, err)
		}
	}

	return nil
}

func (p *Peer) createDataChannels(newRedirect redirect.NewRedirect) error {
	guestTCP, guestUDP, err := newRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create redirects: %w", err)
	}

	if guestTCP != nil {
		if err := p.createDataChannel("tcp", guestTCP); err != nil {
			return fmt.Errorf("failed to create TCP channel: %w", err)
		}
	}

	if guestUDP != nil {
		if err := p.createDataChannel("udp", guestUDP); err != nil {
			return fmt.Errorf("failed to create UDP channel: %w", err)
		}
	}

	return nil
}

func (p *Peer) createDataChannel(label string, redir redirect.Redirect) error {
	dc, err := p.Connection.CreateDataChannel(label, nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %w", label, err)
	}

	// p.mu.Lock()
	// p.dataChannels[label] = dc
	// p.mu.Unlock()

	pipe := NewPipe(dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel",
			"userId", p.UserID,
			"channel", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("DataChannel has closed",
			"userId", p.UserID,
			"channel", label)

		// p.mu.Lock()
		// delete(p.dataChannels, label)
		// p.mu.Unlock()

		pipe.Close()
	})

	return nil
}

func (p *Peer) Terminate() {
	if p.Connection != nil {
		slog.Debug("Closing WebRTC connection", "userId", p.UserID)

		// p.mu.RLock()
		// for label, dc := range p.dataChannels {
		// 	if err := dc.Close(); err != nil {
		// 		slog.Error("Failed to close data channel",
		// 			"userId", p.UserID,
		// 			"channel", label,
		// 			"error", err)
		// 	}
		// }
		// p.mu.RUnlock()

		if err := p.Connection.Close(); err != nil {
			slog.Error("Failed to close WebRTC connection",
				"userId", p.UserID,
				"error", err)
			return
		}
	}
}
