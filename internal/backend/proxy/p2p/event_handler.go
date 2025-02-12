package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/redirect"
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
	AddPeer(peer *Peer)
	GetPeer(peerId string) (*Peer, bool)
	RemovePeer(peerId string)
	CreatePeer(player wire.Player) (*Peer, error)

	Host() (*Peer, bool)
	SetHost(newHostPeer *Peer, newHost wire.Player)
}

type PeerInterface interface {
	GetUserID() string

	SendRTCICECandidate(context.Context, webrtc.ICECandidateInit, string) error
	SendRTCOffer(context.Context, webrtc.SessionDescription, string) error
	SendRTCAnswer(context.Context, webrtc.SessionDescription, string) error
}

type PeerToPeerMessageHandler struct {
	session     PeerInterface
	peerManager PeerManager

	newTCPRedirect redirect.NewRedirect
	newUDPRedirect redirect.NewRedirect
}

func (h *PeerToPeerMessageHandler) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	lgr := slog.With("payload", string(payload), "sessionID", h.session.GetUserID())

	switch eventType {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			lgr.Error(errDecodingJoinRoom, "error", err)
			return err
		}
		return h.handleJoinRoom(ctx, msg.Content)
	case wire.RTCOffer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			lgr.Error(errDecodingRTCOffer, "error", err)
			return err
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCOffer(ctx, msg.Content, msg.From)
	case wire.RTCAnswer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			lgr.Error(errDecodingRTCAnswer, "error", err)
			return err
		}
		if msg.To != h.session.GetUserID() {
			return nil
		}
		return h.handleRTCAnswer(ctx, msg.Content, msg.From)
	case wire.RTCICECandidate:
		_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
		if err != nil {
			lgr.Error(errDecodingRTCCandidate, "error", err)
			return err
		}
		return h.handleRTCCandidate(ctx, msg.Content, msg.From)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			lgr.Error(errDecodingLeaveRoom, "error", err)
			return err
		}
		return h.handleLeaveRoom(ctx, msg.Content)
	case wire.LobbyUsers, wire.JoinLobby, wire.CreateRoom:
		return nil
	case wire.HostMigration:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			lgr.Error(errDecodingJoinRoom, "error", err)
			return err
		}
		return h.handleHostMigration(ctx, msg.Content)
	default:
		lgr.Debug("unknown wire message", "type", eventType.String())
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

	if player.ID() == h.session.GetUserID() {
		slog.Debug("Player is already joined", "userId", player.UserID, "sessionID", h.session.GetUserID())
		return nil
	}

	peer, connected := h.peerManager.GetPeer(player.ID())
	if connected && peer.Connection != nil {
		logger.Debug("Peer already exists, ignoring join", "userId", player.UserID)
		return nil
	}

	logger.Debug("JOIN", "id", player.UserID, "data", player)

	peer, err := h.peerManager.CreatePeer(player)
	if err != nil {
		logger.Warn("Could not add a peer", "userId", player.UserID, "error", err)
		return err
	}

	h.peerManager.AddPeer(peer)

	if err := peer.setupPeerConnection(ctx, h.session, player, true); err != nil {
		return err
	}
	if err := peer.createDataChannels(ctx, h.newTCPRedirect, h.newUDPRedirect, h.session.GetUserID()); err != nil {
		return err
	}

	return nil
}

// handleRTCOffer handles the incoming RTCOffer from another peer.
//
// It sets up the peer connection, creates data channels, and sends the RTCOffer
// to the other peer.
//
// The RTC offer is usually handled by the guest player, who responds to a host.
func (h *PeerToPeerMessageHandler) handleRTCOffer(ctx context.Context, offer wire.Offer, fromUserId string) error {
	logger := slog.With(
		"from", fromUserId,
		"sessionID", h.session.GetUserID(),
	)
	logger.Debug("RTC_OFFER")

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled while handling RTC offer: %w", err)
	}

	peer, found := h.peerManager.GetPeer(offer.Player.ID())
	if !found {
		// logger.Warn("Could not add a peer", "userId", offer.Player.UserID, "error", err)
		return fmt.Errorf("could not add peer: %s", offer.Player.ID())
	}
	// peer, err := h.createPeer(h.session, offer.Player)
	// if err != nil {
	// 	return err
	// }

	if err := peer.setupPeerConnection(ctx, h.session, offer.Player, false); err != nil {
		return err
	}

	peer.Connection.OnDataChannel(func(dc *webrtc.DataChannel) {
		logger.Debug("Data channel opened", "label", dc.Label())

		var redir redirect.Redirect
		var err error
		switch dc.Label() {
		case peer.channelName("tcp", fromUserId, h.session.GetUserID()):
			redir, err = h.newTCPRedirect(peer.Mode, peer.Addr)
			if err != nil {
				logger.Error("Could not create TCP redirect", "error", err)
				return
			}
			peer.PipeTCP = NewPipe(ctx, dc, redir)
		case peer.channelName("udp", fromUserId, h.session.GetUserID()):
			redir, err = h.newUDPRedirect(peer.Mode, peer.Addr)
			if err != nil {
				logger.Error("Could not create UDP redirect", "error", err)
				return
			}
			peer.PipeUDP = NewPipe(ctx, dc, redir)
		default:
			logger.Error("Unknown channel", "label", dc.Label())
			return
		}

		// dc.OnOpen(func() {
		// 	logger.Debug("Data channel opened", "label", dc.Label())
		// 	// dc.SendText("Hello from the server")
		// })
	})

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

// handleRTCAnswer handles the incoming RTCAnswer from another peer.
func (h *PeerToPeerMessageHandler) handleRTCAnswer(ctx context.Context, offer wire.Offer, fromUserId string) error {
	slog.Debug("RTC_ANSWER", "from", fromUserId)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  offer.Offer.SDP,
	}
	peer, ok := h.peerManager.GetPeer(fromUserId)
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

	peer, ok := h.peerManager.GetPeer(fromUserId)
	if !ok {
		return fmt.Errorf("could not find peer %q", fromUserId)
	}

	return peer.Connection.AddICECandidate(candidate)
}

func (h *PeerToPeerMessageHandler) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	slog.Info("Other player is leaving", "playerId", player.ID())

	slog.Debug("LEAVE_ROOM OR LEAVE_LOBBY")

	peer, ok := h.peerManager.GetPeer(player.ID())
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.UserID == h.session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.UserID)
	h.peerManager.RemovePeer(player.ID())
	return nil
}

func (h *PeerToPeerMessageHandler) handleHostMigration(ctx context.Context, newHost wire.Player) error {
	b, _ := json.Marshal(h)
	fmt.Println(string(b))

	oldPeer, ok := h.peerManager.Host()
	if !ok {
		// There is no host, go along.
	} else {
		fmt.Println(oldPeer.Addr)

		// Close connection to the old host
		oldPeer.Terminate()
	}

	// if oldPeer.UserID == h.session.GetUserID() {
	//	// I am the host, not sure what to do
	//	panic("not implemented")
	// }

	newHostPeer, ok := h.peerManager.GetPeer(newHost.ID())
	if !ok {
		panic("could not find peer of new host")
	}

	// todo: write tests
	fmt.Println(newHostPeer.Addr)

	h.peerManager.SetHost(newHostPeer, newHost)

	// response := make([]byte, 8)
	// copy(response[0:4], []byte{1, 0, 0, 0})
	// copy(response[4:], ip.To4())

	return nil
}
