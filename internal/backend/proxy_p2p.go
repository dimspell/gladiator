package backend

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	WebRTCConfig webrtc.Configuration

	Peers map[*Session]*PeersToSessionMapping

	NewRedirect redirect.NewRedirect
}

type PeersToSessionMapping struct {
	Game  *GameRoom
	Peers map[string]*p2p.Peer
}

func NewPeerToPeer() *PeerToPeer {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// {
			// 	URLs: []string{"stun:stun.l.google.com:19302"},
			// },
			// {
			// 	URLs:       []string{"turn:127.0.0.1:3478"},
			// 	Username:   "username1",
			// 	Credential: "password1",
			// },
		},
	}

	return &PeerToPeer{
		// TODO: Test it
		hostIPAddress: net.IPv4(127, 0, 1, 2),

		WebRTCConfig: config,
		Peers:        make(map[*Session]*PeersToSessionMapping),
		NewRedirect:  redirect.New,
	}
}

func (p *PeerToPeer) CreateRoom(params CreateParams, session *Session) (net.IP, error) {
	ipAddr := net.IPv4(127, 0, 0, 1)

	// Create player from session data
	player := wire.Player{
		UserID:      session.UserID,
		Username:    session.Username,
		CharacterID: session.CharacterID,
		ClassType:   byte(session.ClassType),
		IPAddress:   ipAddr.To4().String(),
	}

	gameRoom := NewGameRoom()
	gameRoom.ID = params.GameID
	gameRoom.Name = params.GameID
	gameRoom.SetHost(player)
	gameRoom.SetPlayer(player)

	session.State.SetGameRoom(gameRoom)

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(params HostParams, session *Session) error {
	host := &p2p.Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: p.hostIPAddress},
		Mode:       redirect.CurrentUserIsHost,
	}

	room := session.State.GameRoom()
	if room == nil || room.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	p.Peers[session] = &PeersToSessionMapping{
		Game: room,
		Peers: map[string]*p2p.Peer{
			host.PeerUserID: host,
		},
	}

	if err := p.sendRoomReadyNotification(session, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) sendRoomReadyNotification(session *Session, gameID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return session.SendSetRoomReady(ctx, gameID)
}

func (p *PeerToPeer) GetHostIP(hostIpAddress string, session *Session) net.IP {
	return p.hostIPAddress
}

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *Session) (net.IP, error) {
	// Return the IP address of the player, if he is already in the list.
	mapping, exist := p.Peers[session]
	if exist {
		if peer, ok := mapping.Peers[params.UserID]; ok {
			return peer.Addr.IP, nil
		}
	}

	peer := session.IpRing.NextPeerAddress(
		params.UserID,
		params.UserID == session.GetUserID(),
		params.UserID == params.HostUserID,
	)

	if exist {
		mapping.Peers[peer.PeerUserID] = peer
	} else {
		p.Peers[session] = &PeersToSessionMapping{
			Game:  session.State.GameRoom(),
			Peers: map[string]*p2p.Peer{peer.PeerUserID: peer},
		}
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(params JoinParams, session *Session) (net.IP, error) {
	peer := &p2p.Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:       redirect.None,
	}

	mapping, exist := p.Peers[session]
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers")
	}

	mapping.Peers[peer.PeerUserID] = peer

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *Session) {
	if mapping, exists := p.Peers[session]; exists {
		for _, peer := range mapping.Peers {
			if peer.Connection != nil {
				peer.Connection.Close()
			}
		}
	}
	session.IpRing.Reset()
	delete(p.Peers, session)
}

func (p *PeerToPeer) ExtendWire(session *Session) MessageHandler {
	return &PeerToPeerMessageHandler{
		session: session,
		proxy:   p,
	}
}

type PeerToPeerInterface interface {
	getPeer(session *Session, peerId string) (*p2p.Peer, bool)
	deletePeer(session *Session, peerId string)

	setUpChannels(session *Session, peerId int64, sendRTCOffer bool, createChannels bool) (*p2p.Peer, error)
}

func (p *PeerToPeer) getPeer(session *Session, peerID string) (*p2p.Peer, bool) {
	mapping, ok := p.Peers[session]
	if !ok {
		return nil, false
	}
	peer, ok := mapping.Peers[peerID]
	if !ok {
		return nil, false
	}

	return peer, true
}

func (p *PeerToPeer) setPeer(session *Session, peer *p2p.Peer) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	mapping.Peers[peer.PeerUserID] = peer
}

func (p *PeerToPeer) deletePeer(session *Session, peerID string) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	delete(mapping.Peers, peerID)
}

func (p *PeerToPeer) setUpChannels(session *Session, playerId int64, sendRTCOffer bool, createChannels bool) (*p2p.Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		return nil, err
	}

	gameRoom := session.State.GameRoom()
	player, found := gameRoom.GetPlayer(strconv.FormatInt(playerId, 10))
	if !found {
		return nil, fmt.Errorf("could not find player in game room")
	}

	peer := p.getOrCreatePeer(session, &player)
	peer.Connection = peerConnection

	if !p.isPeerExisting(session, &player) {
		p.setPeer(session, peer)
	}

	if err := p.setupPeerConnection(peerConnection, session, &player, sendRTCOffer); err != nil {
		return nil, err
	}

	if createChannels {
		if err := p.createDataChannels(peerConnection, session, peer); err != nil {
			return nil, err
		}
	}

	return peer, nil
}

func (p *PeerToPeer) getOrCreatePeer(session *Session, player *wire.Player) *p2p.Peer {
	peer, ok := p.getPeer(session, player.ID())
	if !ok {
		isHost := session.State.GameRoom().Host.UserID == player.UserID
		isCurrentUser := session.State.GameRoom().Host.UserID == session.UserID
		return session.IpRing.NextPeerAddress(player.ID(), isCurrentUser, isHost)
	}
	return peer
}

func (p *PeerToPeer) isPeerExisting(session *Session, player *wire.Player) bool {
	_, ok := p.getPeer(session, player.ID())
	return ok
}

func (p *PeerToPeer) setupPeerConnection(peerConnection *webrtc.PeerConnection, session *Session, player *wire.Player, sendRTCOffer bool) error {
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", player.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		reply := wire.ComposeTyped[webrtc.ICECandidateInit](wire.RTCICECandidate, wire.MessageContent[webrtc.ICECandidateInit]{
			From:    session.GetUserID(),
			To:      player.ID(),
			Type:    wire.RTCICECandidate,
			Content: candidate.ToJSON(),
		})
		if err := wire.Write(context.Background(), session.wsConn, reply); err != nil {
			slog.Error("Could not send ICE candidate", "from", session.GetUserID(), "to", player.UserID, "error", err)
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			slog.Error("failed to create offer", "error", err)
			return
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			slog.Error("failed to set local description", "error", err)
			return
		}

		if !sendRTCOffer {
			// If this is a message sent first time after joining,
			// then we send the offer to invite yourself to join other users.
			return
		}

		reply := wire.ComposeTyped[wire.Offer](wire.RTCOffer, wire.MessageContent[wire.Offer]{
			From: session.GetUserID(),
			To:   player.ID(),
			Type: wire.RTCOffer,
			Content: wire.Offer{
				UserID: session.UserID,
				Offer:  offer,
			},
		})
		if err := wire.Write(context.TODO(), session.wsConn, reply); err != nil {
			panic(err)
		}
	})

	return nil
}

func (p *PeerToPeer) createDataChannels(peerConnection *webrtc.PeerConnection, session *Session, peer *p2p.Peer) error {
	roomId := session.State.GameRoom().Name

	if guestTCP, guestUDP, err := p.NewRedirect(peer.Mode, peer.Addr); err == nil {
		if guestTCP != nil {
			if err := p.createDataChannel(peerConnection, fmt.Sprintf("%s/tcp", roomId), guestTCP); err != nil {
				return err
			}
		}

		if guestUDP != nil {
			if err := p.createDataChannel(peerConnection, fmt.Sprintf("%s/udp", roomId), guestUDP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PeerToPeer) createDataChannel(peerConnection *webrtc.PeerConnection, label string, redir redirect.Redirect) error {
	dc, err := peerConnection.CreateDataChannel(label, nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", label, err)
	}

	pipe := p2p.NewPipe(dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel", "label", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("dataChannel has closed", "label", label)

		pipe.Close()
	})

	return nil
}

type PeerToPeerMessageHandler struct {
	session *Session
	proxy   PeerToPeerInterface
}

func (p *PeerToPeerMessageHandler) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)

	switch et {
	case wire.JoinRoom:
		return p.handleJoinRoom(payload)
	case wire.RTCOffer:
		return p.handleRTCOffer(payload)
	case wire.RTCAnswer:
		return p.handleRTCAnswer(payload)
	case wire.RTCICECandidate:
		return p.handleRTCCandidate(payload)
	case wire.LeaveRoom, wire.LeaveLobby:
		return p.handleLeaveRoom(payload)
	default:
		slog.Debug("unknown wire message", slog.String("type", et.String()), slog.String("payload", string(payload)))
		return nil
	}
}

func (p *PeerToPeerMessageHandler) handleJoinRoom(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Player](payload)
	if err != nil {
		slog.Error("failed to decode join room payload", "error", err, "payload", string(payload))
		return nil
	}

	player := msg.Content
	slog.Info("Other player is joining", "playerId", player.ID())
	p.session.State.GameRoom().SetPlayer(player)

	// Validate the message
	if msg.Content.UserID == p.session.UserID {
		return nil
	}

	peer, connected := p.proxy.getPeer(p.session, player.ID())
	if connected && peer.Connection != nil {
		slog.Debug("Peer already exists, ignoring join", "userId", player.UserID)
		return nil
	}

	slog.Debug("JOIN", "id", player.UserID, "data", msg)

	// Add the peer to the list of peers, and start the WebRTC connection
	if _, err := p.proxy.setUpChannels(p.session, player.UserID, true, true); err != nil {
		slog.Warn("Could not add a peer", "userId", player.UserID, "error", err)
		return err
	}

	return nil
}

func (p *PeerToPeerMessageHandler) handleRTCOffer(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Offer](payload)
	if err != nil {
		slog.Error("failed to decode RTC Offer payload", "error", err, "payload", string(payload))
		return nil
	}

	slog.Debug("RTC_OFFER", "from", msg.From, "to", msg.To)

	peer, err := p.proxy.setUpChannels(p.session, msg.Content.UserID, false, false)
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
		From: p.session.GetUserID(),
		To:   msg.From,
		Type: wire.RTCAnswer,
		Content: wire.Offer{
			UserID: p.session.UserID, // TODO: Unused data
			Offer:  answer,
		},
	})
	if err := wire.Write(context.TODO(), p.session.wsConn, response); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (p *PeerToPeerMessageHandler) handleRTCAnswer(payload []byte) error {
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
	peer, ok := p.proxy.getPeer(p.session, msg.From)
	if !ok {
		return fmt.Errorf("could not find peer %q that sent the RTC answer", msg.From)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
}

func (p *PeerToPeerMessageHandler) handleRTCCandidate(payload []byte) error {
	_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
	if err != nil {
		slog.Error("failed to decode RTC ICE Candidate payload", "error", err, "payload", string(payload))
		return nil
	}

	slog.Debug("RTC_ICE_CANDIDATE", "from", msg.From, "to", msg.To)

	peer, ok := p.proxy.getPeer(p.session, msg.From)
	if !ok {
		return fmt.Errorf("could not find peer %q", msg.From)
	}

	return peer.Connection.AddICECandidate(msg.Content)
}

func (p *PeerToPeerMessageHandler) handleLeaveRoom(payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Player](payload)
	if err != nil {
		slog.Error("failed to decode leave-room/leave-lobby payload", "error", err, "payload", string(payload))
		return nil
	}

	player := msg.Content
	slog.Info("Other player is leaving", "playerId", player.ID())
	p.session.State.GameRoom().DeletePlayer(player)

	slog.Debug("LEAVE", "from", msg.From, "to", msg.To)

	peer, ok := p.proxy.getPeer(p.session, msg.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.PeerUserID == p.session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.PeerUserID)
	p.proxy.deletePeer(p.session, msg.From)
	return nil
}
