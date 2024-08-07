package proxy

import (
	"container/ring"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/client"

	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/fxamacker/cbor/v2"
	"github.com/pion/webrtc/v4"
	"golang.org/x/net/websocket"
)

const (
	_ = iota
	ModeHost
	ModeGuest
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	SignalServerURL string
	IpRing          *IpRing

	Peers *client.Peers

	ws *websocket.Conn
	// done chan struct{}
}

func NewPeerToPeer(signalServerURL string) *PeerToPeer {
	return &PeerToPeer{
		SignalServerURL: signalServerURL,
		Peers:           client.NewPeers(),
		IpRing:          NewIpRing(),
	}
}

func (p *PeerToPeer) GetHostIP(hostIpAddress string) net.IP {
	// TODO: Not true, but good enough for now. Joining user will need to have different IP address.
	return net.IPv4(127, 0, 0, 1)
}

func (p *PeerToPeer) Create(params CreateParams) (net.IP, error) {
	if p.ws != nil {
		return nil, fmt.Errorf("already connected to the signal server")
	}
	if err := p.dialSignalServer(params.HostUserID, params.GameID); err != nil {
		return nil, fmt.Errorf("failed to connect to the signal server: %w", err)
	}
	return net.IPv4(127, 0, 0, 1), nil
}

func (p *PeerToPeer) Host(params HostParams) error {
	go p.runWebRTC(ModeHost, params.HostUserID, params.GameID)
	return nil
}

func (p *PeerToPeer) Join(params JoinParams) (net.IP, error) {
	if err := p.dialSignalServer(params.CurrentUserID, params.GameID); err != nil {
		return nil, fmt.Errorf("failed to connect to the signal server: %w", err)
	}

	ip := net.IPv4(127, 0, 1, 2)

	// close(p.done)
	// p.done = make(chan struct{}, 1)

	go p.runWebRTC(ModeGuest, params.CurrentUserID, params.GameID)

	// select {
	// case <-time.After(5 * time.Second):
	// 	return nil, fmt.Errorf("timeout")
	// 	// case ip := <-p.chanMyIP:
	// 	// 	return ip, nil
	// }

	return ip, nil
}

func (p *PeerToPeer) Exchange(params ExchangeParams) (net.IP, error) {
	peer, ok := p.Peers.Get(params.UserID)
	if !ok {
		return nil, fmt.Errorf("user %s not found", params.UserID)
	}
	return peer.IP, nil
}

func (p *PeerToPeer) Close() {
	if p.ws != nil {
		if err := p.ws.Close(); err != nil {
			slog.Warn("Could not close websocket connection", "error", err)
		}
	}
	p.Peers.Reset()
}

func (p *PeerToPeer) dialSignalServer(userId, roomName string) error {
	// Parse the signaling URL provided from the parameters (command flags)
	u, err := url.Parse(p.SignalServerURL)
	if err != nil {
		return err
	}

	// Set parameters
	v := u.Query()
	v.Set("userID", userId)
	v.Set("roomName", roomName)
	u.RawQuery = v.Encode()

	// Connect to the signaling server
	slog.Debug("Connecting to the signaling server", "url", u.String())
	ws, err := websocket.Dial(u.String(), "", "http://localhost:8080")
	if err != nil {
		return err
	}

	// Send "hello" message to the signaling server
	req := &signalserver.Message{
		From:    userId,
		Type:    signalserver.HandshakeRequest,
		Content: roomName,
	}
	if _, err := ws.Write(req.ToCBOR()); err != nil {
		return err
	}

	// Read the response from the signaling server (could be some auth token)
	buf := make([]byte, 128)
	n, err := ws.Read(buf)
	if err != nil {
		slog.Error("Error reading message", "error", err)
		return err
	}
	// Check that the response is a handshake response
	if n == 0 || buf[0] != byte(signalserver.HandshakeResponse) {
		return fmt.Errorf("unexpected handshake response: %v", buf[:n])
	}
	// TODO: Check that the response contains the same room name as the request
	resp, err := decodeCBOR[signalserver.MessageContent[string]](buf[1:n])
	if err != nil {
		return err
	}

	slog.Info("Connected to signaling server", "response", resp.Content)
	p.ws = ws

	return nil
}

func (p *PeerToPeer) sendSignal(message []byte) (err error) {
	if p.ws == nil {
		panic("Not implemented")
	}
	_, err = p.ws.Write(message)
	return
}

func (p *PeerToPeer) runWebRTC(mode int, user string, gameRoom string) {
	signalMessages := make(chan []byte)
	defer func() {
		close(signalMessages)
		p.ws.Close()
	}()

	go func() {
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

	handleSignalMessage := p.handlePackets(mode, user, gameRoom)

	const timeout = time.Second * 25
	timer := time.NewTimer(timeout)
	for {
		resetTimer(timer, timeout)
		select {
		case msg, ok := <-signalMessages:
			if !ok {
				return
			}
			if err := handleSignalMessage(msg); err != nil {
				slog.Error("could not handle signal message", "error", err)
			}
		case <-timer.C:
			if _, err := p.ws.Write([]byte{0}); err != nil {
				// return err
				return
			}
			return
		}
	}
}

func (p *PeerToPeer) handlePackets(mode int, user string, room string) func([]byte) error {
	return func(buf []byte) error {
		switch signalserver.EventType(buf[0]) {
		case signalserver.Join:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Member]) error {
				slog.Debug("JOIN", "id", m.Content.ID)

				// Validate the message
				if m.Content.ID == user {
					// return fmt.Errorf("peer %q is the same as the host, ignoring join", m.Content.ID)
					return nil
				}
				if p.Peers.Exist(m.Content.ID) {
					return fmt.Errorf("peer %q already exists, ignoring join", m.Content.ID)
				}

				// Create a fake endpoint that could be listened and redirect the packets
				guest, err := p.IpRing.CreateClient(mode)
				if err != nil {
					return fmt.Errorf("could not create guest proxy for %s: %v", user, err)
				}

				log.Println("Joining peer", m.Content.ID)

				// Add the peer to the list of peers, and start the WebRTC connection
				member := m.Content
				peer := p.addPeer(member, room, user, guest, true)

				// Create the data channels over the WebRTC connection
				if err := p.createChannels(peer, guest, room); err != nil {
					return fmt.Errorf("could not create data channels: %v", err)
				}
				return nil
			})
		case signalserver.Leave:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[any]) error {
				slog.Debug("LEAVE", "from", m.From, "to", m.To)

				peer, ok := p.Peers.Get(m.From)
				if !ok {
					// fmt.Errorf("could not find peer %q", m.From)
					return nil
				}
				if peer.ID == user {
					return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
				}

				slog.Info("User left", "peer", peer.Name)
				p.Peers.Delete(peer.ID)
				return nil
			})
		case signalserver.RTCOffer:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Offer]) error {
				slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)

				member := signalserver.Member{ID: m.From, Name: m.Content.Name}

				guest, err := p.IpRing.CreateClient(mode)
				if err != nil {
					return fmt.Errorf("could not create guest proxy for %s: %v", user, err)
				}

				peer := p.addPeer(member, room, user, guest, false)

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
					From:    user,
					To:      m.From,
					Type:    signalserver.RTCAnswer,
					Content: signalserver.Offer{Name: peer.Name, Offer: answer},
				}
				if err := p.sendSignal(response.ToCBOR()); err != nil {
					return fmt.Errorf("could not send answer: %v", err)
				}
				return nil
			})
		case signalserver.RTCAnswer:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Offer]) error {
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
			})
		case signalserver.RTCICECandidate:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[webrtc.ICECandidateInit]) error {
				peer, ok := p.Peers.Get(m.From)
				if !ok {
					return fmt.Errorf("could not find peer %q", m.From)
				}
				if err := peer.Connection.AddICECandidate(m.Content); err != nil {
					return fmt.Errorf("could not add ICE candidate: %w", err)
				}
				return nil
			})
		default:
			return nil
		}
	}
}

func (p *PeerToPeer) addPeer(member signalserver.Member, room string, user string, guest client.Proxer, isJoinNotRTCOffer bool) *client.Peer {
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

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	peer := &client.Peer{
		ID:         member.ID,
		Name:       member.Name,
		Connection: peerConnection,
		Proxer:     guest,
	}
	p.Peers.Set(member.ID, peer)

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", member.ID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		reply := &signalserver.Message{
			From:    user,
			To:      member.ID,
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

		if !isJoinNotRTCOffer {
			// If this is a message sent first time after joining,
			// then we send the offer to invite yourself to join other users.
			return
		}

		reply := &signalserver.Message{
			From: user,
			To:   member.ID,
			Type: signalserver.RTCOffer,
			Content: signalserver.Offer{
				Name:  peer.Name,
				Offer: offer,
			},
		}
		if err := p.sendSignal(reply.ToCBOR()); err != nil {
			panic(err)
		}
	})

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnOpen(func() {
			slog.Debug("Opened WebRTC channel", "label", dc.Label(), "peer", member.ID)

			switch {
			case isTCPChannel(dc, room):
				client.NewPipeTCP(dc, guest)
			case isUDPChannel(dc, room):
				client.NewPipeUDP(dc, guest)
			}
		})
	})

	return peer
}

func (p *PeerToPeer) createChannels(peer *client.Peer, other client.Proxer, room string) error {
	// UDP
	dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", room), nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", room, err)
	}
	pipeUDP := client.NewPipeUDP(dcUDP, other)

	dcUDP.OnClose(func() {
		log.Printf("dataChannel for %s has closed", peer.ID)
		p.Peers.Delete(peer.ID)
		pipeUDP.Close()
	})

	// TCP
	dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", room), nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", room, err)
	}
	pipeTCP := client.NewPipeTCP(dcTCP, other)

	dcTCP.OnClose(func() {
		log.Printf("dataChannel for %s has closed", peer.ID)
		p.Peers.Delete(peer.ID)
		pipeTCP.Close()
	})

	return nil
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

type IpRing struct {
	*ring.Ring

	mtx sync.Mutex
}

func NewIpRing() *IpRing {
	r := ring.New(3)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return &IpRing{Ring: r}
}

func (r *IpRing) IP() net.IP {
	d := byte(r.Value.(int))
	defer r.Next()
	return net.IPv4(127, 0, 1, d)
}

func (r *IpRing) CreateClient(mode int) (client.Proxer, error) {
	defer r.mtx.Unlock()
	r.mtx.Lock()

	i := r.Value.(int)
	defer r.Next()

	tcpPort := 7000 + i*2
	udpPort := 7000 + i*2 + 1

	if mode == ModeHost {
		tcpPort += 1000
		udpPort += 1000
	}

	if mode == ModeGuest {
		return client.ListenGuest(
			net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", tcpPort)),
			net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", udpPort)),
		)
	}
	if mode == ModeHost {
		return client.DialHost("127.0.0.1")
	}

	panic("Not implemented")
}
