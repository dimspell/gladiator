package p2p

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"time"

	"github.com/coder/websocket"
	icesignal2 "github.com/dimspell/gladiator/internal/icesignal"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/pion/webrtc/v4"
)

type PeerToPeer struct {
	SignalServerURL string

	RoomName          string
	CurrentUserID     string
	CurrentUserIsHost bool

	IpRing *IpRing
	Peers  *Peers

	ws           *websocket.Conn
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

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Connect to the signaling server
	slog.Debug("Connecting to the signaling server", "url", u.String())
	ws, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		Subprotocols: []string{"signalserver"},
	})
	if err != nil {
		return nil, err
	}

	// Send "hello" message to the signaling server
	req := &icesignal2.Message{
		Type: icesignal2.Hello,
		From: currentUserID,
		Content: icesignal2.Member{
			UserID: currentUserID,
			IsHost: isHost,
		},
	}

	if err := ws.Write(context.TODO(), websocket.MessageText, req.Encode()); err != nil {
		return nil, err
	}

	// Read the response from the signaling server (could be some auth token)
	_, data, err := ws.Read(context.TODO())
	if err != nil {
		slog.Error("Error reading message", "error", err)
		return nil, err
	}
	// Check that the response is a handshake response
	if len(data) == 0 || data[0] != byte(icesignal2.LobbyUsers) {
		return nil, fmt.Errorf("unexpected handshake response: %v", data)
	}
	// TODO: Check that the response contains the same room name as the request
	resp, err := decodeSignalMessage[icesignal2.MessageContent[string]](data[1:])
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

func (p *PeerToPeer) Run(ctx context.Context) {
	signalMessages := make(chan []byte)
	defer func() {
		close(signalMessages)

		if err := p.ws.CloseNow(); err != nil {
			return
		}
	}()

	go func() {
		for {
			_, data, err := p.ws.Read(ctx)
			if err != nil {
				slog.Error("error reading websocket message", "error", err)
				return
			}
			if signalMessages == nil {
				return
			}
			signalMessages <- data
		}
	}()

	// const timeout = time.Second * 25
	// timer := time.NewTimer(timeout)
	for {
		// resetTimer(timer, timeout)

		select {
		case <-ctx.Done():
			slog.Error(ctx.Err().Error())
			return
		case msg, ok := <-signalMessages:
			if !ok {
				return
			}
			if err := p.handlePackets(msg); err != nil {
				slog.Error("could not handle signal message", "error", err)
			}
			// case <-timer.C:
			// 	if _, err := p.ws.Write([]byte{0}); err != nil {
			// 		// return err
			// 		// return
			// 	}
		}
	}
}

func (p *PeerToPeer) handlePackets(buf []byte) error {
	switch icesignal2.EventType(buf[0]) {
	case icesignal2.Join:
		return decodeAndRun(buf[1:], p.handleJoin)
	case icesignal2.Leave:
		return decodeAndRun(buf[1:], p.handleLeave)
	case icesignal2.RTCOffer:
		return decodeAndRun(buf[1:], p.handleRTCOffer)
	case icesignal2.RTCAnswer:
		return decodeAndRun(buf[1:], p.handleRTCAnswer)
	case icesignal2.RTCICECandidate:
		return decodeAndRun(buf[1:], p.handleRTCCandidate)
	default:
		return nil
	}
}

func (p *PeerToPeer) handleJoin(m icesignal2.MessageContent[icesignal2.Member]) error {
	// Validate the message
	if m.Content.UserID == p.CurrentUserID {
		// slog.Debug("Peer is the same as the host, ignoring join", "userId", m.Content.UserID, "host", m.Content.IsHost)
		return nil
	}

	peer, connected := p.Peers.Get(m.Content.UserID)
	if connected && peer.Connection != nil {
		// slog.Debug("Peer already exist, ignoring join", "userId", m.Content.UserID, "host", m.Content.IsHost)
		return nil
	}

	slog.Debug("JOIN", "id", m.Content.UserID, "host", m.Content.IsHost)

	// Add the peer to the list of peers, and start the WebRTC connection
	if _, err := p.addPeer(m.Content, true, true); err != nil {
		slog.Warn("Could not add a peer", "userId", m.Content.UserID, "error", err)
		return err
	}

	return nil
}

func (p *PeerToPeer) handleLeave(m icesignal2.MessageContent[any]) error {
	slog.Debug("LEAVE", "from", m.From, "to", m.To)

	peer, ok := p.Peers.Get(m.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.PeerUserID == p.CurrentUserID {
		// return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
		return nil
	}

	slog.Info("User left", "peer", peer.PeerUserID)
	p.Peers.Delete(peer.PeerUserID)
	return nil
}

func (p *PeerToPeer) handleRTCOffer(m icesignal2.MessageContent[icesignal2.Offer]) error {
	slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)

	peer, err := p.addPeer(m.Content.Member, false, false)
	if err != nil {
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

	response := &icesignal2.Message{
		From: p.CurrentUserID,
		To:   m.From,
		Type: icesignal2.RTCAnswer,
		Content: icesignal2.Offer{
			Member: icesignal2.Member{UserID: p.CurrentUserID, IsHost: p.CurrentUserIsHost}, // TODO: Unused data
			Offer:  answer,
		},
	}
	if err := p.sendSignal(response.Encode()); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (p *PeerToPeer) handleRTCAnswer(m icesignal2.MessageContent[icesignal2.Offer]) error {
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

func (p *PeerToPeer) addPeer(member icesignal2.Member, sendRTCOffer bool, createChannels bool) (*Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		panic(err)
	}

	peer, ok := p.Peers.Get(member.UserID)
	if !ok {
		// TODO: Always guest is created, but should be checked if the user is a host
		peer = p.IpRing.NextPeerAddress(member.UserID, false, false)
	}
	peer.Connection = peerConnection

	p.Peers.Set(member.UserID, peer)

	guestTCP, guestUDP, err := redirect.NewNoop(peer.Mode, peer.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not create guest proxy for %s: %v", member.UserID, err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", member.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		reply := &icesignal2.Message{
			From:    p.CurrentUserID,
			To:      member.UserID,
			Type:    icesignal2.RTCICECandidate,
			Content: candidate.ToJSON(),
		}
		if err := p.sendSignal(reply.Encode()); err != nil {
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

		reply := &icesignal2.Message{
			From: p.CurrentUserID,
			To:   member.UserID,
			Type: icesignal2.RTCOffer,
			Content: icesignal2.Offer{
				Member: icesignal2.Member{
					UserID: p.CurrentUserID,
					IsHost: p.CurrentUserIsHost,
				}, // TODO: Is it correct?
				Offer: offer,
			},
		}
		if err := p.sendSignal(reply.Encode()); err != nil {
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

	if createChannels {
		if guestTCP != nil {
			dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", p.RoomName), nil)
			if err != nil {
				return nil, fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
			}
			pipeTCP := NewPipe(dcTCP, guestTCP)

			dcTCP.OnClose(func() {
				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
				p.Peers.Delete(peer.PeerUserID)
				pipeTCP.Close()
			})
		}

		if guestUDP != nil {
			// UDP
			dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", p.RoomName), nil)
			if err != nil {
				return nil, fmt.Errorf("could not create data channel %q: %v", p.RoomName, err)
			}
			pipeUDP := NewPipe(dcUDP, guestUDP)

			dcUDP.OnClose(func() {
				log.Printf("dataChannel for %s has closed", peer.PeerUserID)
				p.Peers.Delete(peer.PeerUserID)
				pipeUDP.Close()
			})
		}
	}

	return peer, nil
}

func (p *PeerToPeer) handleRTCCandidate(m icesignal2.MessageContent[webrtc.ICECandidateInit]) error {
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

func (p *PeerToPeer) sendSignal(message []byte) error {
	slog.Debug(string(message))
	return p.ws.Write(context.TODO(), websocket.MessageText, message)
}

func decodeAndRun[T any](data []byte, f func(T) error) error {
	v, err := decodeSignalMessage[T](data)
	if err != nil {
		return err
	}
	return f(v)
}

func decodeSignalMessage[T any](data []byte) (v T, err error) {
	err = icesignal2.DefaultCodec.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding signal message", "error", err, "payload", string(data))
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

// func resetTimer(t *time.Timer, d time.Duration) {
// 	if !t.Stop() {
// 		select {
// 		case <-t.C:
// 		default:
// 		}
// 	}
// 	t.Reset(d)
// }

func (p *PeerToPeer) Close() {
	if p.ws != nil {
		if err := p.ws.CloseNow(); err != nil {
			slog.Warn("Could not close websocket connection", "error", err)
		}
	}

	p.Peers.Reset()
}
