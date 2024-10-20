package backend

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"log/slog"
// 	"net"
//
// 	"github.com/dimspell/gladiator/internal/proxy/p2p"
// 	"github.com/dimspell/gladiator/internal/proxy/redirect"
// 	"github.com/dimspell/gladiator/internal/wire"
// 	"github.com/pion/webrtc/v4"
// )
//
// var _ Proxy = (*PeerToPeer)(nil)
//
// type PeerToPeer struct {
// 	// A custom IP address to which we will connect to.
// 	hostIPAddress net.IP
//
// 	WebRTCConfig webrtc.Configuration
// }
//
// func NewPeerToPeer() *PeerToPeer {
// 	config := webrtc.Configuration{
// 		ICEServers: []webrtc.ICEServer{
// 			// {
// 			// 	URLs: []string{"stun:stun.l.google.com:19302"},
// 			// },
// 			// {
// 			// 	URLs:       []string{"turn:127.0.0.1:3478"},
// 			// 	Username:   "username1",
// 			// 	Credential: "password1",
// 			// },
// 		},
// 	}
//
// 	return &PeerToPeer{
// 		hostIPAddress: net.IPv4(127, 0, 1, 2),
// 		WebRTCConfig:  config,
// 	}
// }
//
// func (p *PeerToPeer) CreateRoom(params CreateParams, session *Session) (net.IP, error) {
// 	return net.IPv4(127, 0, 0, 1), nil
// }
//
// func (p *PeerToPeer) HostRoom(params HostParams, session *Session) (err error) {
// 	host := &p2p.Peer{
// 		PeerUserID: session.GetUserID(),
// 		Addr:       &redirect.Addressing{IP: p.hostIPAddress},
// 		Mode:       redirect.CurrentUserIsHost,
// 	}
// 	session.RoomPlayers.Set(host.PeerUserID, host)
// 	return nil
// }
//
// func (p *PeerToPeer) GetHostIP(_ string, session *Session) net.IP { return p.hostIPAddress }
//
// func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *Session) (net.IP, error) {
// 	// Return the IP address of the player, if he is already in the list.
// 	for _, peer := range p.gatheredPeers {
// 		if peer.PeerUserID == params.UserID {
// 			return peer.Addr.IP, nil
// 		}
// 	}
//
// 	peer := session.IpRing.NextPeerAddress(
// 		params.UserID,
// 		params.UserID == session.GetUserID(),
// 		params.UserID == params.HostUserID,
// 	)
// 	p.gatheredPeers = append(p.gatheredPeers, peer)
//
// 	return peer.Addr.IP, nil
// }
//
// func (p *PeerToPeer) Join(params JoinParams, session *Session) (err error) {
// 	current := &p2p.Peer{
// 		PeerUserID: session.GetUserID(),
// 		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
// 		Mode:       redirect.None,
// 	}
// 	p.RoomPlayers.Set(current.PeerUserID, current)
//
// 	return err
// }
//
// func (p *PeerToPeer) Close(session *Session) {
// 	session.IpRing = p2p.NewIpRing()
// 	session.RoomPlayers = p2p.NewPeers()
// }
//
// func (p *PeerToPeer) ExtendWire(ctx context.Context, session *Session, et wire.EventType, payload []byte) {
// 	var err error
// 	switch et {
// 	case wire.JoinRoom:
// 		err = wire.DecodeAndRun(payload, p.handleJoinRoom, session)
// 	case wire.LeaveRoom:
// 		err = wire.DecodeAndRun(payload, p.handleLeaveRoom, session)
// 	case wire.RTCOffer:
// 		err = wire.DecodeAndRun(payload, p.handleRTCOffer, session)
// 	case wire.RTCAnswer:
// 		err = wire.DecodeAndRun(payload, p.handleRTCAnswer, session)
// 	case wire.RTCICECandidate:
// 		err = wire.DecodeAndRun(payload, p.handleRTCCandidate, session)
// 	default:
// 		//	Ignore
// 	}
// 	if err != nil {
// 		slog.Debug("failed to decode wire", slog.String("type", et.String()), slog.String("payload", string(payload)))
// 	}
// }
//
// func (p *PeerToPeer) handleJoinRoom(m wire.MessageContent[wire.Member], session *Session) error {
// 	// Validate the message
// 	if m.Content.UserID == session.GetUserID() {
// 		return nil
// 	}
//
// 	peer, connected := session.RoomPlayers.Get(m.Content.UserID)
// 	if connected && peer.Connection != nil {
// 		// slog.Debug("Peer already exist, ignoring join", "userId", m.Content.UserID, "host", m.Content.IsHost)
// 		return nil
// 	}
//
// 	slog.Debug("JOIN", "id", m.Content.UserID, "host", m.Content.IsHost)
//
// 	// Add the peer to the list of peers, and start the WebRTC connection
// 	if _, err := p.addPeer(session, m.Content, true, true); err != nil {
// 		slog.Warn("Could not add a peer", "userId", m.Content.UserID, "error", err)
// 		return err
// 	}
//
// 	return nil
// }
//
// func (p *PeerToPeer) handleLeaveRoom(m wire.MessageContent[any], session *Session) error {
// 	slog.Debug("LEAVE", "from", m.From, "to", m.To)
//
// 	peer, ok := session.RoomPlayers.Get(m.From)
// 	if !ok {
// 		// fmt.Errorf("could not find peer %q", m.From)
// 		return nil
// 	}
// 	if peer.PeerUserID == session.GetUserID() {
// 		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
// 		return nil
// 	}
//
// 	slog.Info("User left", "peer", peer.PeerUserID)
// 	session.RoomPlayers.Delete(peer.PeerUserID)
// 	return nil
// }
//
// func (p *PeerToPeer) handleRTCOffer(m wire.MessageContent[wire.Offer], session *Session) error {
// 	slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)
//
// 	peer, err := p.addPeer(session, m.Content.Member, false, false)
// 	if err != nil {
// 		return err
// 	}
//
// 	if err := peer.Connection.SetRemoteDescription(m.Content.Offer); err != nil {
// 		return fmt.Errorf("could not set remote description: %v", err)
// 	}
//
// 	answer, err := peer.Connection.CreateAnswer(nil)
// 	if err != nil {
// 		return fmt.Errorf("could not create answer: %v", err)
// 	}
//
// 	if err := peer.Connection.SetLocalDescription(answer); err != nil {
// 		return fmt.Errorf("could not set local description: %v", err)
// 	}
//
// 	response := &wire.Message{
// 		From: session.GetUserID(),
// 		To:   m.From,
// 		Type: wire.RTCAnswer,
// 		Content: wire.Offer{
// 			Member: wire.Member{UserID: session.GetUserID(), IsHost: p.CurrentUserIsHost}, // TODO: Unused data
// 			Offer:  answer,
// 		},
// 	}
//
// 	if err := wire.EncodeAndWrite(context.TODO(), session.wsConn, response); err != nil {
// 		return fmt.Errorf("could not send answer: %v", err)
// 	}
// 	return nil
// }
//
// func (p *PeerToPeer) handleRTCAnswer(m wire.MessageContent[wire.Offer], session *Session) error {
// 	slog.Debug("RTC_ANSWER", "from", m.From, "to", m.To)
//
// 	answer := webrtc.SessionDescription{
// 		Type: webrtc.SDPTypeAnswer,
// 		SDP:  m.Content.Offer.SDP,
// 	}
// 	peer, ok := session.RoomPlayers.Get(m.From)
// 	if !ok {
// 		return fmt.Errorf("could not find peer %q that sent the RTC answer", m.From)
// 	}
// 	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
// 		return fmt.Errorf("could not set remote description: %v", err)
// 	}
// 	return nil
// }
//
// func (p *PeerToPeer) addPeer(session *Session, member wire.Member, sendRTCOffer bool, createChannels bool) (*p2p.Peer, error) {
// 	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	peer, ok := session.RoomPlayers.Get(member.UserID)
// 	if !ok {
// 		// TODO: Always guest is created, but should be checked if the user is a host
// 		peer = session.IpRing.NextPeerAddress(member.UserID, false, false)
// 	}
// 	peer.Connection = peerConnection
//
// 	session.RoomPlayers.Set(member.UserID, peer)
//
// 	guestTCP, guestUDP, err := redirect.NewNoop(peer.Mode, peer.Addr)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not create guest proxy for %s: %v", member.UserID, err)
// 	}
//
// 	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
// 		slog.Debug("ICE Connection State has changed", "peer", member.UserID, "state", connectionState.String())
// 	})
//
// 	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
// 		if candidate == nil {
// 			return
// 		}
// 		reply := &wire.Message{
// 			From:    session.GetUserID(),
// 			To:      member.UserID,
// 			Type:    wire.RTCICECandidate,
// 			Content: candidate.ToJSON(),
// 		}
// 		if err := wire.EncodeAndWrite(context.TODO(), session.wsConn, reply); err != nil {
// 			panic(err)
// 		}
// 	})
//
// 	peerConnection.OnNegotiationNeeded(func() {
// 		offer, err := peerConnection.CreateOffer(nil)
// 		if err != nil {
// 			panic(err)
// 		}
//
// 		if err := peerConnection.SetLocalDescription(offer); err != nil {
// 			panic(err)
// 		}
//
// 		if !sendRTCOffer {
// 			// If this is a message sent first time after joining,
// 			// then we send the offer to invite yourself to join other users.
// 			return
// 		}
//
// 		reply := &wire.Message{
// 			From: session.GetUserID(),
// 			To:   member.UserID,
// 			Type: wire.RTCOffer,
// 			Content: wire.Offer{
// 				Member: wire.Member{
// 					UserID: session.GetUserID(),
// 					IsHost: p.CurrentUserIsHost,
// 				}, // TODO: Is it correct?
// 				Offer: offer,
// 			},
// 		}
// 		if err := wire.EncodeAndWrite(context.TODO(), session.wsConn, reply); err != nil {
// 			panic(err)
// 		}
// 	})
//
// 	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
// 		dc.OnOpen(func() {
// 			slog.Debug("Opened WebRTC channel", "label", dc.Label(), "peer", member.UserID)
//
// 			switch {
// 			case isTCPChannel(dc, p.RoomName):
// 				p2p.NewPipe(dc, guestTCP)
// 			case isUDPChannel(dc, p.RoomName):
// 				p2p.NewPipe(dc, guestUDP)
// 			}
// 		})
// 	})
//
// 	if createChannels {
// 		if guestTCP != nil {
// 			dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", p.RoomName), nil)
// 			if err != nil {
// 				return nil, fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
// 			}
// 			pipeTCP := p2p.NewPipe(dcTCP, guestTCP)
//
// 			dcTCP.OnClose(func() {
// 				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
// 				session.RoomPlayers.Delete(peer.PeerUserID)
// 				pipeTCP.Close()
// 			})
// 		}
//
// 		if guestUDP != nil {
// 			// UDP
// 			dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", p.RoomName), nil)
// 			if err != nil {
// 				return nil, fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
// 			}
// 			pipeUDP := p2p.NewPipe(dcUDP, guestUDP)
//
// 			dcUDP.OnClose(func() {
// 				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
// 				session.RoomPlayers.Delete(peer.PeerUserID)
// 				pipeUDP.Close()
// 			})
// 		}
// 	}
//
// 	return peer, nil
// }
//
// func (p *PeerToPeer) handleRTCCandidate(m wire.MessageContent[webrtc.ICECandidateInit], session *Session) error {
// 	slog.Debug("RTC_ICE_CANDIDATE", "from", m.From, "to", m.To)
//
// 	peer, ok := session.RoomPlayers.Get(m.From)
// 	if !ok {
// 		return fmt.Errorf("could not find peer %q", m.From)
// 	}
// 	if err := peer.Connection.AddICECandidate(m.Content); err != nil {
// 		return fmt.Errorf("could not add ICE candidate: %w", err)
// 	}
// 	return nil
// }
