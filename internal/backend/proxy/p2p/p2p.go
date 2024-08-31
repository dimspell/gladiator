package p2p

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"time"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/fxamacker/cbor/v2"
	"github.com/pion/webrtc/v4"
	"golang.org/x/net/websocket"
)

type PeerToPeer struct {
	SignalServerURL string

	RoomName          string
	CurrentUserID     string
	CurrentUserIsHost bool

	IpRing *IpRing
	Peers  *Peers

	ws           WebSocket
	WebRTCConfig webrtc.Configuration
}

func DialSignalServer(signalServerURL string, currentUserID, roomName string, isHost bool) (*PeerToPeer, error) {
	// Parse the signaling URL provided from the parameters (command flags)
	u, err := url.Parse(signalServerURL)
	if err != nil {
		return nil, err
	}

	// Set parameters
	v := u.Query()
	v.Set("userID", currentUserID)
	v.Set("roomName", roomName)
	u.RawQuery = v.Encode()

	// Connect to the signaling server
	slog.Debug("Connecting to the signaling server", "url", u.String())
	ws, err := websocket.Dial(u.String(), "", "http://localhost:8080")
	if err != nil {
		return nil, err
	}

	// Send "hello" message to the signaling server
	req := &signalserver.Message{
		Type: signalserver.HandshakeRequest,
		From: currentUserID,
		Content: signalserver.Member{
			UserID: currentUserID,
			IsHost: isHost,
			Joined: isHost, // Note: Host is always joined.
		},
	}
	if _, err := ws.Write(req.ToCBOR()); err != nil {
		return nil, err
	}

	// Read the response from the signaling server (could be some auth token)
	buf := make([]byte, 128)
	n, err := ws.Read(buf)
	if err != nil {
		slog.Error("Error reading message", "error", err)
		return nil, err
	}
	// Check that the response is a handshake response
	if n == 0 || buf[0] != byte(signalserver.HandshakeResponse) {
		return nil, fmt.Errorf("unexpected handshake response: %v", buf[:n])
	}
	// TODO: Check that the response contains the same room name as the request
	resp, err := decodeCBOR[signalserver.MessageContent[string]](buf[1:n])
	if err != nil {
		return nil, err
	}

	slog.Info("Connected to signaling server", "response", resp.Content)

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
		SignalServerURL: signalServerURL,
		IpRing:          NewIpRing(),
		Peers:           NewPeers(),
		WebRTCConfig:    config,
		ws:              ws,

		CurrentUserID:     currentUserID,
		CurrentUserIsHost: isHost,
		RoomName:          roomName,
	}, nil
}

func (p *PeerToPeer) Run(hostUserID string) {
	signalMessages := make(chan []byte)
	defer func() {
		if err := p.ws.Close(); err != nil {
			return
		}
	}()

	go func() {
		defer close(signalMessages)

		for {
			buf := make([]byte, 1024)
			n, err := p.ws.Read(buf)
			if err != nil {
				slog.Error("error reading websocket message", "error", err)
				return
			}
			if signalMessages == nil {
				return
			}
			signalMessages <- buf[:n]
		}
	}()

	const timeout = time.Second * 25
	timer := time.NewTimer(timeout)
	for {
		resetTimer(timer, timeout)

		select {
		case msg, ok := <-signalMessages:
			if !ok {
				return
			}
			if err := p.handlePackets(msg); err != nil {
				slog.Error("could not handle signal message", "error", err)
			}
		case <-timer.C:
			if _, err := p.ws.Write([]byte{0}); err != nil {
				// return err
				// return
			}
		}
	}
}

func (p *PeerToPeer) handlePackets(buf []byte) error {
	switch signalserver.EventType(buf[0]) {
	case signalserver.Join:
		return decodeAndRun(buf[1:], p.handleJoin)
	case signalserver.Leave:
		return decodeAndRun(buf[1:], p.handleLeave)
	case signalserver.RTCOffer:
		return decodeAndRun(buf[1:], p.handleRTCOffer)
	case signalserver.RTCAnswer:
		return decodeAndRun(buf[1:], p.handleRTCAnswer)
	case signalserver.RTCICECandidate:
		return decodeAndRun(buf[1:], p.handleRTCCandidate)
	default:
		return nil
	}
}

func (p *PeerToPeer) handleJoin(m signalserver.MessageContent[signalserver.Member]) error {
	// Validate the message
	if m.Content.UserID == p.CurrentUserID {
		// slog.Debug("Peer is the same as the host, ignoring join", "userId", m.Content.UserID, "host", m.Content.IsHost)
		return nil
	}
	if p.Peers.Exist(m.Content.UserID) {
		// slog.Debug("Peer already exist, ignoring join", "userId", m.Content.UserID, "host", m.Content.IsHost)
		return nil
	}

	slog.Debug("JOIN", "id", m.Content.UserID, "host", m.Content.IsHost)

	// Create a fake endpoint that could be listened and redirect the packets
	guestTCP, guestUDP, err := p.IpRing.CreateClient(p.CurrentUserIsHost, m.Content)
	if err != nil {
		return fmt.Errorf("could not create guest proxy for %s: %v", p.CurrentUserID, err)
	}

	log.Println("Joining peer", m.Content.UserID)

	// Add the peer to the list of peers, and start the WebRTC connection
	member := m.Content
	peer := p.addPeer(member, guestTCP, guestUDP, true)

	if guestTCP != nil {
		dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", p.RoomName), nil)
		if err != nil {
			return fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
		}
		pipeTCP := NewPipe(dcTCP, guestTCP)

		dcTCP.OnClose(func() {
			log.Printf("dataChannel for %s has closed", peer.User.UserID)
			p.Peers.Delete(peer.User.UserID)
			pipeTCP.Close()
		})
	}

	if guestUDP != nil {
		// UDP
		dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", p.RoomName), nil)
		if err != nil {
			return fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
		}
		pipeUDP := NewPipe(dcUDP, guestUDP)

		dcUDP.OnClose(func() {
			log.Printf("dataChannel for %s has closed", peer.User.UserID)
			p.Peers.Delete(peer.User.UserID)
			pipeUDP.Close()
		})
	}

	return nil
}

func (p *PeerToPeer) handleLeave(m signalserver.MessageContent[any]) error {
	slog.Debug("LEAVE", "from", m.From, "to", m.To)

	peer, ok := p.Peers.Get(m.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.User.UserID == p.CurrentUserID {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.User.UserID)
	p.Peers.Delete(peer.User.UserID)
	return nil
}

func (p *PeerToPeer) handleRTCOffer(m signalserver.MessageContent[signalserver.Offer]) error {
	slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)

	guestTCP, guestUDP, err := p.IpRing.CreateClient(p.CurrentUserIsHost, m.Content.Member)
	if err != nil {
		return fmt.Errorf("could not create guest proxy for %s: %v", m.From, err)
	}

	peer := p.addPeer(m.Content.Member, guestTCP, guestUDP, false)

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

	response := &signalserver.Message{
		From: p.CurrentUserID,
		To:   m.From,
		Type: signalserver.RTCAnswer,
		Content: signalserver.Offer{
			Member: signalserver.Member{UserID: p.CurrentUserID, IsHost: p.CurrentUserIsHost}, // TODO: Unused data
			Offer:  answer,
		},
	}
	if err := p.sendSignal(response.ToCBOR()); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (p *PeerToPeer) handleRTCAnswer(m signalserver.MessageContent[signalserver.Offer]) error {
	slog.Debug("RTC_ANSWER", "from", m.From, "to", m.To)

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  m.Content.Offer.SDP,
	}
	peer, ok := p.Peers.Get(m.From)
	if !ok {
		return fmt.Errorf("could not find peer %q that sent the RTC answer", m.From)
	}
	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("could not set remote description: %v", err)
	}
	return nil
}

func (p *PeerToPeer) addPeer(member signalserver.Member, guestTCP Redirector, guestUDP Redirector, sendRTCOffer bool) *Peer {
	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		panic(err)
	}

	peer := &Peer{
		User:       member,
		Connection: peerConnection,
	}
	p.Peers.Set(member.UserID, peer)

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", member.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		reply := &signalserver.Message{
			From:    p.CurrentUserID,
			To:      member.UserID,
			Type:    signalserver.RTCICECandidate,
			Content: candidate.ToJSON(),
		}
		if err := p.sendSignal(reply.ToCBOR()); err != nil {
			panic(err)
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

		reply := &signalserver.Message{
			From: p.CurrentUserID,
			To:   member.UserID,
			Type: signalserver.RTCOffer,
			Content: signalserver.Offer{
				Member: signalserver.Member{
					UserID: p.CurrentUserID,
					IsHost: p.CurrentUserIsHost,
				}, // TODO: Is it correct?
				Offer: offer,
			},
		}
		if err := p.sendSignal(reply.ToCBOR()); err != nil {
			panic(err)
		}
	})

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			slog.Debug("Opened WebRTC channel", "label", dc.Label(), "peer", member.UserID)

			switch {
			case isTCPChannel(dc, p.RoomName):
				NewPipe(dc, guestTCP)
			case isUDPChannel(dc, p.RoomName):
				NewPipe(dc, guestUDP)
			}
		})
	})

	return peer
}

func (p *PeerToPeer) handleRTCCandidate(m signalserver.MessageContent[webrtc.ICECandidateInit]) error {
	slog.Debug("RTC_ICE_CANDIDATE", "from", m.From, "to", m.To)

	peer, ok := p.Peers.Get(m.From)
	if !ok {
		return fmt.Errorf("could not find peer %q", m.From)
	}
	if err := peer.Connection.AddICECandidate(m.Content); err != nil {
		return fmt.Errorf("could not add ICE candidate: %w", err)
	}
	return nil
}

func (p *PeerToPeer) sendSignal(message []byte) (err error) {
	if p.ws == nil {
		panic("Not implemented")
	}
	_, err = p.ws.Write(message)
	return
}

func decodeAndRun[T any](data []byte, f func(T) error) error {
	v, err := decodeCBOR[T](data)
	if err != nil {
		return err
	}
	return f(v)
}

func decodeCBOR[T any](data []byte) (v T, err error) {
	err = cbor.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding CBOR", "error", err, "payload", string(data))
		panic(err)
	}
	return v, err
}

func isTCPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/tcp", room)
}

func isUDPChannel(dc *webrtc.DataChannel, room string) bool {
	return dc.Label() == fmt.Sprintf("%s/udp", room)
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

func (p *PeerToPeer) Close() {
	if p.ws != nil {
		if err := p.ws.Close(); err != nil {
			slog.Warn("Could not close websocket connection", "error", err)
		}
	}
	p.Peers.Reset()
}
