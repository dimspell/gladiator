package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type PeerManager interface {
	GetPeer(session *bsession.Session, peerId string) (*Peer, bool)
	RemovePeer(session *bsession.Session, peerId string)
}

type PeerToPeerMessageHandler struct {
	session       *bsession.Session
	peerManager   PeerManager
	setUpChannels func(session *bsession.Session, player wire.Player, sendRTCOffer bool, createChannels bool) (*Peer, error)
}

func (h *PeerToPeerMessageHandler) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	switch eventType {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Error("failed to decode join room payload", "error", err, "payload", string(payload))
			return nil
		}
		return h.handleJoinRoom(ctx, msg.Content)
	case wire.RTCOffer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error("failed to decode RTC Offer payload", "error", err, "payload", string(payload))
			return nil
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCOffer(ctx, msg.Content, msg.From)
	case wire.RTCAnswer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error("failed to decode RTC Answer payload", "error", err, "payload", string(payload))
			return nil
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCAnswer(ctx, msg.Content, msg.From)
	case wire.RTCICECandidate:
		_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
		if err != nil {
			slog.Error("failed to decode RTC ICE Candidate payload", "error", err, "payload", string(payload))
			return nil
		}
		return h.handleRTCCandidate(ctx, msg.Content, msg.From)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Error("failed to decode leave-room/leave-lobby payload", "error", err, "payload", string(payload))
			return nil
		}
		return h.handleLeaveRoom(ctx, msg.Content)
	default:
		slog.Debug("unknown wire message", slog.String("type", eventType.String()), slog.String("payload", string(payload)))
		return nil
	}
}

func (h *PeerToPeerMessageHandler) handleJoinRoom(ctx context.Context, player wire.Player) error {
	slog.Info("Other player is joining", "playerId", player.ID())

	// Validate the message
	if player.UserID == h.session.UserID {
		return nil
	}

	peer, connected := h.peerManager.GetPeer(h.session, player.ID())
	if connected && peer.Connection != nil {
		slog.Debug("Peer already exists, ignoring join", "userId", player.UserID)
		return nil
	}

	slog.Debug("JOIN", "id", player.UserID, "data", player)

	// Add the peer to the list of peers, and start the WebRTC connection
	if _, err := h.setUpChannels(h.session, player, true, true); err != nil {
		slog.Warn("Could not add a peer", "userId", player.UserID, "error", err)
		return err
	}

	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCOffer(ctx context.Context, offer wire.Offer, fromUserId string) error {
	slog.Debug("RTC_OFFER", "from", fromUserId)

	peer, err := h.setUpChannels(h.session, offer.Player, false, false)
	if err != nil {
		return err
	}

	if err := peer.Connection.SetRemoteDescription(offer.Offer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}

	answer, err := peer.Connection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %v", err)
	}

	if err := peer.Connection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %v", err)
	}

	if err := h.session.SendRTCAnswer(ctx, answer, fromUserId); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCAnswer(ctx context.Context, offer wire.Offer, fromUserId string) error {
	slog.Debug("RTC_ANSWER", "from", fromUserId)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  offer.Offer.SDP,
	}
	peer, ok := h.peerManager.GetPeer(h.session, fromUserId)
	if !ok {
		return fmt.Errorf("could not find peer %q that sent the RTC answer", fromUserId)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCCandidate(ctx context.Context, candidate webrtc.ICECandidateInit, fromUserId string) error {
	slog.Debug("RTC_ICE_CANDIDATE", "from", fromUserId)

	peer, ok := h.peerManager.GetPeer(h.session, fromUserId)
	if !ok {
		return fmt.Errorf("could not find peer %q", fromUserId)
	}

	return peer.Connection.AddICECandidate(candidate)
}

func (h *PeerToPeerMessageHandler) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	slog.Info("Other player is leaving", "playerId", player.ID())

	slog.Debug("LEAVE_ROOM OR LEAVE_LOBBY")

	peer, ok := h.peerManager.GetPeer(h.session, player.ID())
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.UserID == h.session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.UserID)
	h.peerManager.RemovePeer(h.session, player.ID())
	return nil
}
