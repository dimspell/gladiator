package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type PeerManager interface {
	AddPeer(peer *Peer)
	GetPeer(peerId int64) (*Peer, bool)
	RemovePeer(peerId int64)
	CreatePeer(player wire.Player) (*Peer, error)

	Host() (*Peer, bool)
	SetHost(newHostPeer *Peer, newHost wire.Player)
}

type PeerInterface interface {
	GetUserID() int64

	SendRTCICECandidate(ctx context.Context, candidate webrtc.ICECandidateInit, recipientId int64) error
	SendRTCOffer(ctx context.Context, offer webrtc.SessionDescription, recipientId int64) error
	SendRTCAnswer(ctx context.Context, offer webrtc.SessionDescription, recipientId int64) error
}

type PeerToPeerMessageHandler struct {
	session     PeerInterface
	peerManager PeerManager

	newTCPRedirect redirect.NewRedirect
	newUDPRedirect redirect.NewRedirect
}

// Handle is a generic function to handle event messages from the WebSocket.
//
// The sequence of events in a WebRTC connection establishment is crucial:
//
// 1. offer creation,
// 2. offer setting (local description),
// 3. offer sending,
// 4. answer receiving,
// 5. answer setting (remote description),
// 6. ICE candidate exchange.
func (h *PeerToPeerMessageHandler) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)
	userID := h.session.GetUserID()

	logger := slog.With("payload", string(payload), "sessionID", userID)

	switch eventType {
	case wire.LobbyUsers, wire.JoinLobby, wire.CreateRoom:
		return nil
	case wire.JoinRoom:
		return decodeAndHandle(ctx, logger, payload, wire.JoinRoom.String(), h.handleJoinRoom)
	case wire.LeaveRoom, wire.LeaveLobby:
		return decodeAndHandle(ctx, logger, payload, wire.LeaveRoom.String(), h.handleLeaveRoom)
	case wire.HostMigration:
		return decodeAndHandle(ctx, logger, payload, wire.HostMigration.String(), h.handleHostMigration)
	case wire.RTCOffer:
		return handleRTCMessage(ctx, logger, payload, wire.RTCOffer.String(), userID, h.handleRTCOffer)
	case wire.RTCAnswer:
		return handleRTCMessage(ctx, logger, payload, wire.RTCAnswer.String(), userID, h.handleRTCAnswer)
	case wire.RTCICECandidate:
		return handleRTCMessage(ctx, logger, payload, wire.RTCICECandidate.String(), userID, h.handleRTCCandidate)
	default:
		logger.Debug("unknown wire message", "type", eventType.String())
		return nil
	}
}

const errDecodingPayload = "failed to decode payload for event: %s"

// Generic handler for simple event messages
func decodeAndHandle[T any](ctx context.Context, logger *slog.Logger, payload []byte, eventName string, handler func(context.Context, T) error) error {
	_, msg, err := wire.DecodeTyped[T](payload)
	if err != nil {
		logger.Error(fmt.Sprintf(errDecodingPayload, eventName), "error", err)
		return err
	}
	return handler(ctx, msg.Content)
}

// Generic handler for RTC messages
func handleRTCMessage[T any](ctx context.Context, logger *slog.Logger, payload []byte, eventName string, userID int64, handler func(context.Context, T, int64) error) error {
	_, msg, err := wire.DecodeTyped[T](payload)
	if err != nil {
		logger.Error(fmt.Sprintf(errDecodingPayload, eventName), "error", err)
		return err
	}

	if msg.To != strconv.FormatInt(userID, 10) {
		return nil
	}

	fromUserID, err := strconv.ParseInt(msg.From, 10, 64)
	if err != nil || fromUserID <= 0 {
		return err
	}

	return handler(ctx, msg.Content, fromUserID)
}

func (h *PeerToPeerMessageHandler) handleJoinRoom(ctx context.Context, player wire.Player) error {
	logger := slog.With("playerId", player.ID(), "userID", h.session.GetUserID())

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled while handling join room: %w", err)
	}

	if player.UserID == h.session.GetUserID() {
		slog.Debug("Player is already joined")
		return nil
	}

	peer, connected := h.peerManager.GetPeer(player.UserID)
	if connected && peer.Connection != nil {
		logger.Debug("Peer already exists, ignoring join")
		return nil
	}

	logger.Info("New player joining")

	peer, err := h.peerManager.CreatePeer(player)
	if err != nil {
		logger.Warn("Could not add a peer", "error", err)
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
func (h *PeerToPeerMessageHandler) handleRTCOffer(ctx context.Context, offer wire.Offer, fromUserID int64) error {
	logger := slog.With("from", fromUserID, "sessionID", h.session.GetUserID())
	logger.Debug("Processing RTC_OFFER")

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled while handling RTC offer: %w", err)
	}

	peer, found := h.peerManager.GetPeer(offer.Player.UserID)
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

	// <-webrtc.GatheringCompletePromise(peer.Connection)

	peer.Connection.OnDataChannel(func(dc *webrtc.DataChannel) {
		logger.Debug("Data channel opened", "label", dc.Label())

		var redir redirect.Redirect
		var err error
		switch dc.Label() {
		case peer.channelName("tcp", fromUserID, h.session.GetUserID()):
			redir, err = h.newTCPRedirect(peer.Mode, peer.Addr)
			if err != nil {
				logger.Error("Could not create TCP redirect", "error", err)
				return
			}
			peer.PipeTCP = NewPipe(ctx, dc, redir)
		case peer.channelName("udp", fromUserID, h.session.GetUserID()):
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

	if err := h.session.SendRTCAnswer(ctx, answer, fromUserID); err != nil {
		return fmt.Errorf("could not send answer: %w", err)
	}
	return nil
}

// handleRTCAnswer handles the incoming RTCAnswer from another peer.
//
// The RTC answer is usually handled by the host, who received a message from
// the guest player.
func (h *PeerToPeerMessageHandler) handleRTCAnswer(ctx context.Context, offer wire.Offer, fromUserID int64) error {
	logger := slog.With("from", fromUserID, "sessionID", h.session.GetUserID())
	logger.Debug("Processing RTC_ANSWER", "from", fromUserID)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  offer.Offer.SDP,
	}
	peer, ok := h.peerManager.GetPeer(fromUserID)
	if !ok {
		return fmt.Errorf("could not find peer %d", fromUserID)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
}

func (h *PeerToPeerMessageHandler) handleRTCCandidate(ctx context.Context, candidate webrtc.ICECandidateInit, fromUserId int64) error {
	slog.Debug("RTC_ICE_CANDIDATE", "from", fromUserId)

	peer, ok := h.peerManager.GetPeer(fromUserId)
	if !ok {
		return fmt.Errorf("could not find peer %d", fromUserId)
	}

	return peer.Connection.AddICECandidate(candidate)
}

func (h *PeerToPeerMessageHandler) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	slog.Info("Other player is leaving", "playerId", player.ID())

	slog.Debug("LEAVE_ROOM OR LEAVE_LOBBY")

	peer, ok := h.peerManager.GetPeer(player.UserID)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.UserID == h.session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.UserID)
	h.peerManager.RemovePeer(player.UserID)
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

	newHostPeer, ok := h.peerManager.GetPeer(newHost.UserID)
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
