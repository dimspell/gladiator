package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

const (
	errDecodingJoinRoom     = "failed to decode join room payload"
	errDecodingRTCOffer     = "failed to decode RTC Offer payload"
	errDecodingRTCAnswer    = "failed to decode RTC Answer payload"
	errDecodingRTCCandidate = "failed to decode RTC ICE Candidate payload"
	errDecodingLeaveRoom    = "failed to decode leave-room/leave-lobby payload"
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
			slog.Error(errDecodingJoinRoom, "error", err, "payload", string(payload), "sessionID", h.session.GetUserID())
			return err
		}
		return h.handleJoinRoom(ctx, msg.Content)
	case wire.RTCOffer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error(errDecodingRTCOffer, "error", err, "payload", string(payload), "sessionID", h.session.GetUserID())
			return err
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCOffer(ctx, msg.Content, msg.From)
	case wire.RTCAnswer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error(errDecodingRTCAnswer, "error", err, "payload", string(payload), "sessionID", h.session.GetUserID())
			return err
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCAnswer(ctx, msg.Content, msg.From)
	case wire.RTCICECandidate:
		_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
		if err != nil {
			slog.Error(errDecodingRTCCandidate, "error", err, "payload", string(payload),
				"sessionID", h.session.GetUserID())
			return err
		}
		return h.handleRTCCandidate(ctx, msg.Content, msg.From)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Error(errDecodingLeaveRoom, "error", err, "payload", string(payload), "sessionID", h.session.GetUserID())
			return err
		}
		return h.handleLeaveRoom(ctx, msg.Content)
	case wire.LobbyUsers, wire.JoinLobby:
		return nil
	default:
		slog.Debug("unknown wire message", "type", eventType.String(), "payload", string(payload), "sessionID", h.session.GetUserID())
		return nil
	}
}

func (h *PeerToPeerMessageHandler) handleJoinRoom(ctx context.Context, player wire.Player) error {
	logger := slog.With(
		"playerId", player.ID(),
		"sessionID", h.session.GetUserID(),
	)
	logger.Info("Other player is joining")

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled while handling join room: %w", err)
	}

	if player.UserID == h.session.UserID {
		return nil
	}

	peer, connected := h.peerManager.GetPeer(h.session, player.ID())
	if connected && peer.Connection != nil {
		logger.Debug("Peer already exists, ignoring join", "userId", player.UserID)
		return nil
	}

	logger.Debug("JOIN", "id", player.UserID, "data", player)

	peer, err := h.setUpChannels(h.session, player, true, true)
	if err != nil {
		logger.Warn("Could not add a peer", "userId", player.UserID, "error", err)
		return err
	}

	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCOffer(ctx context.Context, offer wire.Offer, fromUserId string) error {
	logger := slog.With(
		"from", fromUserId,
		"sessionID", h.session.GetUserID(),
	)
	logger.Debug("RTC_OFFER")

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled while handling RTC offer: %w", err)
	}

	peer, err := h.setUpChannels(h.session, offer.Player, false, false)
	if err != nil {
		return err
	}

	if err := peer.Connection.SetRemoteDescription(offer.Offer); err != nil {
		return fmt.Errorf("could not set remote description: %w", err)
	}

	answer, err := peer.Connection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %w", err)
	}

	if err := peer.Connection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %w", err)
	}

	if err := h.session.SendRTCAnswer(ctx, answer, fromUserId); err != nil {
		return fmt.Errorf("could not send answer: %w", err)
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
