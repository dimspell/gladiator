package proxy

import (
	"container/ring"
	"fmt"
	"io"
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

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	SignalServerURL string
	IpRing          *IpRing

	Peers *client.Peers

	ws WebSocket
	// done chan struct{}
}

type Role string

type WebSocket interface {
	io.Closer
	io.Reader
	io.Writer
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
	if err := p.dialSignalServer(params.HostUserID, params.GameID, Role(signalserver.RoleHost)); err != nil {
		return nil, fmt.Errorf("failed to connect to the signal server: %w", err)
	}
	return net.IPv4(127, 0, 0, 1), nil
}

func (p *PeerToPeer) Host(params HostParams) error {
	go p.runWebRTC(Role(signalserver.RoleHost), params.HostUserID, params.GameID)
	return nil
}

func (p *PeerToPeer) Join(params JoinParams) (net.IP, error) {
	if err := p.dialSignalServer(params.CurrentUserID, params.GameID, Role(signalserver.RoleGuest)); err != nil {
		return nil, fmt.Errorf("failed to connect to the signal server: %w", err)
	}

	ip := net.IPv4(127, 0, 1, 2)

	// close(p.done)
	// p.done = make(chan struct{}, 1)

	go p.runWebRTC(Role(signalserver.RoleGuest), params.CurrentUserID, params.GameID)

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

func (p *PeerToPeer) dialSignalServer(userId, roomName string, role Role) error {
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
		Type: signalserver.HandshakeRequest,
		From: userId,
		Content: signalserver.Member{
			UserID: userId,
			Role:   string(role),
		},
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

func (p *PeerToPeer) runWebRTC(mode Role, user string, gameRoom string) {
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
				// return
			}
		}
	}
}

func (p *PeerToPeer) handlePackets(currentUserRole Role, currentUserId string, room string) func([]byte) error {
	return func(buf []byte) error {
		switch signalserver.EventType(buf[0]) {
		case signalserver.Join:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Member]) error {
				slog.Debug("JOIN", "id", m.Content.UserID, "role", m.Content.Role)
				return p.handleJoin(m, currentUserId, room, currentUserRole)
			})
		case signalserver.Leave:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[any]) error {
				slog.Debug("LEAVE", "from", m.From, "to", m.To)
				return p.handleLeave(m, currentUserId)
			})
		case signalserver.RTCOffer:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Offer]) error {
				slog.Debug("RTC_OFFER", "from", m.From, "to", m.To)
				return p.handleRTCOffer(m, currentUserId, room, currentUserRole)
			})
		case signalserver.RTCAnswer:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Offer]) error {
				slog.Debug("RTC_ANSWER", "from", m.From, "to", m.To)
				return p.handleRTCAnswer(m, currentUserId, room, currentUserRole)
			})
		case signalserver.RTCICECandidate:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[webrtc.ICECandidateInit]) error {
				slog.Debug("RTC_ICE_CANDIDATE", "from", m.From, "to", m.To)
				return p.handleRTCCandidate(m)
			})
		default:
			return nil
		}
	}
}

func (p *PeerToPeer) handleJoin(m signalserver.MessageContent[signalserver.Member], currentUserId, room string, currentUserRole Role) error {
	// Validate the message
	if m.Content.UserID == currentUserId {
		slog.Debug("Peer is the same as the host, ignoring join", "userId", m.Content.UserID, "role", m.Content.Role)
		return nil
	}
	if p.Peers.Exist(m.Content.UserID) {
		slog.Debug("Peer already exist, ignoring join", "userId", m.Content.UserID, "role", m.Content.Role)
		return nil
	}

	// Create a fake endpoint that could be listened and redirect the packets
	guest, err := p.IpRing.CreateClient(currentUserRole, m.Content)
	if err != nil {
		return fmt.Errorf("could not create guest proxy for %s: %v", currentUserId, err)
	}

	log.Println("Joining peer", m.Content.UserID)

	// Add the peer to the list of peers, and start the WebRTC connection
	member := m.Content
	peer := p.addPeer(member, room, currentUserId, currentUserRole, guest, true)

	// Create the data channels over the WebRTC connection
	if err := p.createChannels(peer, guest, room); err != nil {
		return fmt.Errorf("could not create data channels: %v", err)
	}
	return nil
}

func (p *PeerToPeer) handleLeave(m signalserver.MessageContent[any], user string) error {
	peer, ok := p.Peers.Get(m.From)
	if !ok {
		// fmt.Errorf("could not find peer %q", m.From)
		return nil
	}
	if peer.User.UserID == user {
		return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
	}

	slog.Info("User left", "peer", peer.User.UserID)
	p.Peers.Delete(peer.User.UserID)
	return nil
}

func (p *PeerToPeer) handleRTCOffer(m signalserver.MessageContent[signalserver.Offer], currentUserId string, room string, currentUserRole Role) error {
	guest, err := p.IpRing.CreateClient(currentUserRole, m.Content.Member)
	if err != nil {
		return fmt.Errorf("could not create guest proxy for %s: %v", currentUserId, err)
	}

	peer := p.addPeer(m.Content.Member, room, currentUserId, currentUserRole, guest, false)

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
		From: currentUserId,
		To:   m.From,
		Type: signalserver.RTCAnswer,
		Content: signalserver.Offer{
			Member: signalserver.Member{UserID: currentUserId, Role: string(currentUserRole)}, // TODO: Unused data
			Offer:  answer,
		},
	}
	if err := p.sendSignal(response.ToCBOR()); err != nil {
		return fmt.Errorf("could not send answer: %v", err)
	}
	return nil
}

func (p *PeerToPeer) handleRTCAnswer(m signalserver.MessageContent[signalserver.Offer], user string, room string, mode Role) error {
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

func (p *PeerToPeer) addPeer(member signalserver.Member, room string, user string, role Role, guest client.Proxer, sendRTCOffer bool) *client.Peer {
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
		User: member,
		// IP:         unknown, TODO: Fix me
		Proxer:     guest,
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
			From:    user,
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
			From: user,
			To:   member.UserID,
			Type: signalserver.RTCOffer,
			Content: signalserver.Offer{
				Member: signalserver.Member{
					UserID: user,
					Role:   string(role),
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
		log.Printf("dataChannel for %s has closed", peer.User.UserID)
		p.Peers.Delete(peer.User.UserID)
		pipeUDP.Close()
	})

	// TCP
	dcTCP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/tcp", room), nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", room, err)
	}
	pipeTCP := client.NewPipeTCP(dcTCP, other)

	dcTCP.OnClose(func() {
		log.Printf("dataChannel for %s has closed", peer.User.UserID)
		p.Peers.Delete(peer.User.UserID)
		pipeTCP.Close()
	})

	return nil
}

func (p *PeerToPeer) handleRTCCandidate(m signalserver.MessageContent[webrtc.ICECandidateInit]) error {
	peer, ok := p.Peers.Get(m.From)
	if !ok {
		return fmt.Errorf("could not find peer %q", m.From)
	}
	if err := peer.Connection.AddICECandidate(m.Content); err != nil {
		return fmt.Errorf("could not add ICE candidate: %w", err)
	}
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
	Ring *ring.Ring
	mtx  sync.Mutex
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

func (r *IpRing) NextInt() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	d := r.Ring.Value.(int)
	r.Ring = r.Ring.Next()
	return d
}

func (r *IpRing) NextIP() net.IP {
	return net.IPv4(127, 0, 1, byte(r.NextInt()))
}

func (r *IpRing) NextPort() string {
	return fmt.Sprintf("2137%d", r.NextInt())
}

func (r *IpRing) CreateClient(currentUserRole Role, member signalserver.Member) (proxer client.Proxer, err error) {
	if currentUserRole == Role(signalserver.RoleHost) {
		return client.DialHost("127.0.0.1")
	}

	i := r.NextInt()
	tcpPort := 7000 + i*2
	udpPort := 7000 + i*2 + 1

	return client.ListenGuest(
		net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", tcpPort)),
		net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", udpPort)),
	)
}
