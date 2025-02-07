package proxy

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type Peer struct {
	// UserID uniquely identifies the peer
	UserID string

	// Addr contains network addressing information
	Addr *redirect.Addressing

	// Mode defines the operating mode of the peer
	Mode redirect.Mode

	// Connection holds the WebRTC peer connection
	Connection *webrtc.PeerConnection

	Connected chan struct{}

	PipeTCP *Pipe
	PipeUDP *Pipe
}

func NewPeer(connection *webrtc.PeerConnection, r *IpRing, userId string, isCurrentUser, isHost bool) (*Peer, error) {

	switch true {
	case isCurrentUser:
		return &Peer{
			UserID:     userId,
			Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:       redirect.CurrentUserIsHost,
			Connection: connection,
		}, nil
	case !isCurrentUser && isHost:
		ip, portTCP, portUDP, err := r.NextAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to get next address: %w", err)
		}
		return &Peer{
			UserID:     userId,
			Addr:       &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP},
			Mode:       redirect.OtherUserIsHost,
			Connection: connection,
		}, nil
	case !isCurrentUser && !isHost:
		ip, _, portUDP, err := r.NextAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to get next address: %w", err)
		}
		return &Peer{
			UserID:     userId,
			Addr:       &redirect.Addressing{IP: ip, TCPPort: "", UDPPort: portUDP},
			Mode:       redirect.OtherUserHasJoined,
			Connection: connection,
		}, nil
	default:
		return &Peer{
			UserID:     userId,
			Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
			Mode:       redirect.OtherUserIsJoining,
			Connection: connection,
		}, nil
	}
}

func (p *Peer) setupPeerConnection(ctx context.Context, session *bsession.Session, player wire.Player, sendRTCOffer bool) error {
	slog.Debug("setting up peer connection", "userId", player.UserID)

	p.Connection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed",
			"userId", player.UserID,
			"state", connectionState.String())
	})
	p.Connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			p.Connected <- struct{}{}
		case webrtc.PeerConnectionStateDisconnected:
			slog.Error("Peer connection disconnected", "userId", player.UserID)
			p.Terminate()
		}
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

func (p *Peer) handleNegotiation(ctx context.Context, session *bsession.Session, player wire.Player, sendRTCOffer bool) error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer for peer %d: %w", player.UserID, err)
	}

	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description for peer %d: %w", player.UserID, err)
	}

	if sendRTCOffer {
		if err := session.SendRTCOffer(ctx, offer, player.ID()); err != nil {
			return fmt.Errorf("failed to send RTC offer to peer %d: %w", player.UserID, err)
		}
	}

	return nil
}

func (p *Peer) createDataChannels(ctx context.Context, newTCPRedirect, newUDPRedirect redirect.NewRedirect, myUserId string) error {
	guestTCP, err := newTCPRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create TCP redirect: %w", err)
	}
	guestUDP, err := newUDPRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP redirect: %w", err)
	}

	if guestTCP != nil {
		pipe, err := p.createDataChannel(ctx, p.channelName("tcp", myUserId, p.UserID), guestTCP)
		if err != nil {
			return fmt.Errorf("failed to create TCP channel: %w", err)
		}
		p.PipeTCP = pipe
	}

	if guestUDP != nil {
		pipe, err := p.createDataChannel(ctx, p.channelName("udp", myUserId, p.UserID), guestUDP)
		if err != nil {
			return fmt.Errorf("failed to create UDP channel: %w", err)
		}
		p.PipeUDP = pipe
	}

	return nil
}

func (p *Peer) channelName(proto string, from, to string) string {
	return fmt.Sprintf("/redirect/proto/%s/user/%s/to/%s", proto, from, to)
}

func (p *Peer) createDataChannel(ctx context.Context, label string, redir redirect.Redirect) (*Pipe, error) {
	dc, err := p.Connection.CreateDataChannel(label, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create data channel %q: %w", label, err)
	}

	// p.mu.Lock()
	// p.dataChannels[label] = dc
	// p.mu.Unlock()

	log.Println("Created data channel", "userId", p.UserID, "channel", label)
	pipe := NewPipe(ctx, dc, redir)

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

	return pipe, nil
}

func (p *Peer) Terminate() {
	slog.Debug("Terminating peer connection", "userId", p.UserID)

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
