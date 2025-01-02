package backend

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type PeerToPeerInterface interface {
	getPeer(session *Session, peerId string) (*p2p.Peer, bool)
	deletePeer(session *Session, peerId string)

	setUpChannels(session *Session, peerId int64, sendRTCOffer bool, createChannels bool) (*p2p.Peer, error)
}

type PeerToPeerMessageHandler struct {
	session *Session
	proxy   PeerToPeerInterface
}

func (h *PeerToPeerMessageHandler) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	switch eventType {
	case wire.JoinRoom:
		return h.handleJoinRoom(payload)
	case wire.RTCOffer:
		return h.handleRTCOffer(payload)
	case wire.RTCAnswer:
		return h.handleRTCAnswer(payload)
	case wire.RTCICECandidate:
		return h.handleRTCCandidate(payload)
	case wire.LeaveRoom, wire.LeaveLobby:
		return h.handleLeaveRoom(payload)
	default:
		slog.Debug("unknown wire message", slog.String("type", eventType.String()), slog.String("payload", string(payload)))
		return nil
	}
}

func (h *PeerToPeerMessageHandler) handleJoinRoom(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Player](payload)
	if err != nil {
		slog.Error("failed to decode join room payload", "error", err, "payload", string(payload))
		return nil
	}

	player := msg.Content
	slog.Info("Other player is joining", "playerId", player.ID())
	h.session.State.GameRoom().SetPlayer(player)

	// Validate the message
	if msg.Content.UserID == h.session.UserID {
		return nil
	}

	peer, connected := h.proxy.getPeer(h.session, player.ID())
	if connected && peer.Connection != nil {
		slog.Debug("Peer already exists, ignoring join", "userId", player.UserID)
		return nil
	}

	slog.Debug("JOIN", "id", player.UserID, "data", msg)

	// Add the peer to the list of peers, and start the WebRTC connection
	if _, err := h.proxy.setUpChannels(h.session, player.UserID, true, true); err != nil {
		slog.Warn("Could not add a peer", "userId", player.UserID, "error", err)
		return err
	}

	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCOffer(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Offer](payload)
	if err != nil {
		slog.Error("failed to decode RTC Offer payload", "error", err, "payload", string(payload))
		return nil
	}

	slog.Debug("RTC_OFFER", "from", msg.From, "to", msg.To)

	peer, err := h.proxy.setUpChannels(h.session, msg.Content.UserID, false, false)
	if err != nil {
		return err
	}

	if err := peer.Connection.SetRemoteDescription(msg.Content.Offer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}

	answer, err := peer.Connection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("could not create answer: %v", err)
	}

	if err := peer.Connection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("could not set local description: %v", err)
	}

	response := wire.ComposeTyped[wire.Offer](wire.RTCAnswer, wire.MessageContent[wire.Offer]{
		From: h.session.GetUserID(),
		To:   msg.From,
		Type: wire.RTCAnswer,
		Content: wire.Offer{
			UserID: h.session.UserID, // TODO: Unused data
			Offer:  answer,
		},
	})
	if err := wire.Write(context.TODO(), h.session.wsConn, response); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCAnswer(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Offer](payload)
	if err != nil {
		slog.Error("failed to decode RTC Answer payload", "error", err, "payload", string(payload))
		return nil
	}

	slog.Debug("RTC_ANSWER", "from", msg.From, "to", msg.To)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.Content.Offer.SDP,
	}
	peer, ok := h.proxy.getPeer(h.session, msg.From)
	if !ok {
		return fmt.Errorf("could not find peer %q that sent the RTC answer", msg.From)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCCandidate(payload []byte) error {
	_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
	if err != nil {
		slog.Error("failed to decode RTC ICE Candidate payload", "error", err, "payload", string(payload))
		return nil
	}

	slog.Debug("RTC_ICE_CANDIDATE", "from", msg.From, "to", msg.To)

	peer, ok := h.proxy.getPeer(h.session, msg.From)
	if !ok {
		return fmt.Errorf("could not find peer %q", msg.From)
	}

	return peer.Connection.AddICECandidate(msg.Content)
}

func (h *PeerToPeerMessageHandler) handleLeaveRoom(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Player](payload)
	if err != nil {
		slog.Error("failed to decode leave-room/leave-lobby payload", "error", err, "payload", string(payload))
		return nil
	}

	player := msg.Content
	slog.Info("Other player is leaving", "playerId", player.ID())
	h.session.State.GameRoom().DeletePlayer(player)

	slog.Debug("LEAVE", "from", msg.From, "to", msg.To)

	peer, ok := h.proxy.getPeer(h.session, msg.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.PeerUserID == h.session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.PeerUserID)
	h.proxy.deletePeer(h.session, msg.From)
	return nil
}
