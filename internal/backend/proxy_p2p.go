package backend

import (
	"context"
	"fmt"
	"log"
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

	// Deprecated: unused
	Games map[string]*GameRoom

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

func (p *PeerToPeer) HostRoom(params HostParams, session *Session) (err error) {
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

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
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
	session.IpRing.Reset()
	delete(p.Peers, session)
}

func (p *PeerToPeer) ExtendWire(ctx context.Context, session *Session, et wire.EventType, payload []byte) {
	// slog.Debug("Received extend wire event", "session", session, "payload", payload, "event", et.String())

	switch et {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Error("failed to decode join room payload", "error", err, "payload", string(payload))
			return
		}

		player := msg.Content
		slog.Info("Other player is joining", "playerId", player.ID())

		session.State.GameRoom().SetPlayer(player)

		if err := p.handleJoinRoom(msg, session); err != nil {
			slog.Warn("Failed to join room", "error", err, "session", session)
			return
		}
	case wire.RTCOffer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error("failed to decode RTC Offer payload", "error", err, "payload", string(payload))
			return
		}

		if err := p.handleRTCOffer(msg, session); err != nil {
			slog.Warn("Failed to handle RTC Offer", "error", err, "session", session)
			return
		}
	case wire.RTCAnswer:
		_, msg, err := wire.DecodeTyped[wire.Offer](payload)
		if err != nil {
			slog.Error("failed to decode RTC Answer payload", "error", err, "payload", string(payload))
			return
		}

		if err := p.handleRTCAnswer(msg, session); err != nil {
			slog.Warn("Failed to handle rtc answer", "error", err, "session", session)
			return
		}
	case wire.RTCICECandidate:
		_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
		if err != nil {
			slog.Error("failed to decode RTC ICE Candidate payload", "error", err, "payload", string(payload))
			return
		}

		if err := p.handleRTCCandidate(msg, session); err != nil {
			slog.Warn("RTCC init error", "error", err, "session", session)
			return
		}
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Error("failed to decode leave-room/leave-lobby payload", "error", err, "payload", string(payload))
			return
		}

		player := msg.Content
		slog.Info("Other player is leaving", "playerId", player.ID())

		session.State.GameRoom().DeletePlayer(player)

		if err := p.handleLeaveRoom(msg, session); err != nil {
			slog.Warn("Failed to leave room", "error", err, "playerId", player.ID(), "session", session)
			return
		}
	default:
		//	Ignore
		slog.Debug("unknown wire message", slog.String("type", et.String()), slog.String("payload", string(payload)))
	}
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

func (p *PeerToPeer) handleJoinRoom(m wire.MessageContent[wire.Player], session *Session) error {
	// Validate the message
	if m.Content.UserID == session.UserID {
		return nil
	}

	peer, connected := p.getPeer(session, m.Content.ID())
	if connected && peer.Connection != nil {
		slog.Debug("Peer already exist, ignoring join", "userId", m.Content.UserID)
		return nil
	}

	slog.Debug("JOIN", "id", m.Content.UserID, "data", m)

	// Add the peer to the list of peers, and start the WebRTC connection
	if _, err := p.setUpChannels(session, m.Content.UserID, true, true); err != nil {
		slog.Warn("Could not add a peer", "userId", m.Content.UserID, "error", err)
		return err
	}

	return nil
}

func (p *PeerToPeer) handleLeaveRoom(m wire.MessageContent[wire.Player], session *Session) error {
	slog.Debug("LEAVE", "from", m.From, "to", m.To)

	peer, ok := p.getPeer(session, m.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.PeerUserID == session.GetUserID() {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.PeerUserID)
	p.deletePeer(session, m.From)
	return nil
}

func (p *PeerToPeer) handleRTCOffer(m wire.MessageContent[wire.Offer], session *Session) error {
	slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)

	peer, err := p.setUpChannels(session, m.Content.UserID, false, false)
	if err != nil {
		panic(err)
		return err
	}

	if err := peer.Connection.SetRemoteDescription(m.Content.Offer); err != nil {
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
		From: session.GetUserID(),
		To:   m.From,
		Type: wire.RTCAnswer,
		Content: wire.Offer{
			UserID: session.UserID, // TODO: Unused data
			Offer:  answer,
		},
	})
	if err := wire.Write(context.TODO(), session.wsConn, response); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (p *PeerToPeer) handleRTCAnswer(m wire.MessageContent[wire.Offer], session *Session) error {
	slog.Debug("RTC_ANSWER", "from", m.From, "to", m.To)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  m.Content.Offer.SDP,
	}
	peer, ok := p.getPeer(session, m.From)
	if !ok {
		return fmt.Errorf("could not find peer %q that sent the RTC answer", m.From)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
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

	peer, ok := p.getPeer(session, player.ID())
	if !ok {
		isHost := gameRoom.Host.UserID == player.UserID
		isCurrentUser := gameRoom.Host.UserID == session.UserID
		peer = session.IpRing.NextPeerAddress(player.ID(), isCurrentUser, isHost)
	}
	peer.Connection = peerConnection

	if !ok {
		p.setPeer(session, peer)
	}

	slog.Debug("Setting up redirect", "user", session.UserID, "mode", peer.Mode, "addr", peer.Addr)
	guestTCP, guestUDP, err := p.NewRedirect(peer.Mode, peer.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not create guest proxy for %d: %v", player.UserID, err)
	}

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
		if err := wire.Write(context.TODO(), session.wsConn, reply); err != nil {
			slog.Error("Could not send ICE candidate", "from", session.GetUserID(), "to", player.UserID, "error", err)
			// TODO: Here likely is the connection closed, so it can't be sent further
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
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

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			slog.Debug("Opened WebRTC channel", "label", dc.Label(), "peer", player.UserID)

			switch {
			case isTCPChannel(dc, session.State.GameRoom().ID):
				p2p.NewPipe(dc, guestTCP)
			case isUDPChannel(dc, session.State.GameRoom().ID):
				p2p.NewPipe(dc, guestUDP)
			}
		})
	})

	if createChannels {
		roomId := session.State.GameRoom().Name

		if guestTCP != nil {
			dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", roomId), nil)
			if err != nil {
				return nil, fmt.Errorf("could not create data channel %q: %v", roomId, err)
			}
			pipeTCP := p2p.NewPipe(dcTCP, guestTCP)

			dcTCP.OnClose(func() {
				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
				// session.RoomPlayers.Delete(peer.PeerUserID)
				pipeTCP.Close()
			})
		}

		if guestUDP != nil {
			// UDP
			dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", roomId), nil)
			if err != nil {
				return nil, fmt.Errorf("could not create data channel %q: %v", roomId, err)
			}
			pipeUDP := p2p.NewPipe(dcUDP, guestUDP)

			dcUDP.OnClose(func() {
				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
				// session.RoomPlayers.Delete(peer.PeerUserID)
				pipeUDP.Close()
			})
		}
	}

	return peer, nil
}

func (p *PeerToPeer) handleRTCCandidate(m wire.MessageContent[webrtc.ICECandidateInit], session *Session) error {
	slog.Debug("RTC_ICE_CANDIDATE", "from", m.From, "to", m.To)

	peer, ok := p.getPeer(session, m.From)
	if !ok {
		return fmt.Errorf("could not find peer %q", m.From)
	}
	if err := peer.Connection.AddICECandidate(m.Content); err != nil {
		return fmt.Errorf("could not add ICE candidate: %w", err)
	}
	return nil
}

func isTCPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/tcp", room)
}

func isUDPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/udp", room)
}
