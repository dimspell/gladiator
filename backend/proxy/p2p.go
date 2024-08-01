package proxy

import (
	"container/ring"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/dimspell/gladiator/backend/proxy/client"
	"github.com/dimspell/gladiator/console/signalserver"
	"github.com/fxamacker/cbor/v2"
	"github.com/pion/webrtc/v4"
	"golang.org/x/net/websocket"
	"golang.org/x/sync/errgroup"
)

// TODO: Remove me and refactor me.
const todoRoomNameDefault = "room"

const (
	ModeNone = iota
	ModeHost
	ModeGuest
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	SignalServerURL string
	IpRing          *IpRing

	mode  int
	Peers *client.Peers
	ws    *websocket.Conn

	host   *client.HostListener
	guests [3]*client.GuestProxy
}

func NewPeerToPeer(signalServerURL string) *PeerToPeer {
	return &PeerToPeer{
		SignalServerURL: signalServerURL,
		IpRing:          NewIpRing(),
		Peers:           client.NewPeers(),
	}
}

func (p *PeerToPeer) GetHostIP(hostIpAddress string) net.IP {
	return net.IPv4(127, 0, 0, 1)
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

func (p *PeerToPeer) Create(localIPAddress string, hostUser string) (net.IP, error) {
	if p.ws != nil {
		panic("Not implemented")
	}
	if err := p.dialSignalServer(hostUser, todoRoomNameDefault); err != nil {
		return nil, err
	}

	return p.GetHostIP(""), nil
}

// HostGame connects to the game host and redirects the traffic to the P2P
// network. The game host is expected to be running on the same machine.
func (p *PeerToPeer) HostGame(gameRoom GameRoom, currentUser User) error {
	ip := p.GetHostIP("")
	host, err := client.ListenHost(ip.To4().String())
	if err != nil {
		return err
	}
	go p.runWebRTC(ModeHost, User(currentUser), GameRoom(gameRoom), ip, host)
	return nil
}

func (p *PeerToPeer) Join(gameId string, hostUser string, currentPlayer string, ipAddress string) (net.IP, error) {
	ip := p.IpRing.IP()
	guest, err := client.NewGuestProxy(ip.To4().String())
	if err != nil {
		return nil, err
	}
	go p.runWebRTC(ModeGuest, User(currentPlayer), GameRoom(gameId), ip, guest)
	return ip, nil
}

func (p *PeerToPeer) Exchange(gameId string, userId string, ipAddress string) (net.IP, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PeerToPeer) Close() {
	if p == nil {
		return
	}
	if p.ws != nil {
		if err := p.ws.Close(); err != nil {
			slog.Warn("Could not close websocket connection", "error", err)
		}
	}
}

func (p *PeerToPeer) runWebRTC(mode int, user User, gameRoom GameRoom, ip net.IP, proxy any) {
	if p.ws == nil {
		panic("Not implemented")
	}

	signalMessages := make(chan []byte)
	g, ctx := errgroup.WithContext(context.TODO())

	handlePacket := p.handlePackets(mode, user, gameRoom, ip)

	g.Go(func() error {
		resetTimer := func(t *time.Timer, d time.Duration) {
			if !t.Stop() {
				select {
				case <-t.C:
				default:
				}
			}
			t.Reset(d)
		}
		defer func() {
			close(signalMessages)
		}()

		const timeout = time.Second * 25
		timer := time.NewTimer(timeout)
		for {
			resetTimer(timer, timeout)
			select {
			case msg := <-signalMessages:
				// if err := p.ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				// 	return err
				// }
				if err := handlePacket(ctx, msg); err != nil {
					log.Println("handle packet", err)
				}
			case <-timer.C:
				// if err := p.ws.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				// 	return err
				// }
				if _, err := p.ws.Write([]byte{0}); err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	g.Go(func() error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// if err := p.ws.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			// 	return err
			// }

			buf := make([]byte, 1024)
			n, err := p.ws.Read(buf)
			if err != nil {
				return fmt.Errorf("error reading message: %v", err)
			}

			signalMessages <- buf[:n]
		}
	})

	if err := g.Wait(); err != nil {
		panic(err)
	}
}

func (p *PeerToPeer) handlePackets(mode int, user User, room GameRoom, ip net.IP) func(context.Context, []byte) error {
	return func(ctx context.Context, buf []byte) error {
		switch signalserver.EventType(buf[0]) {
		case signalserver.Join:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Member]) error {
				// Validate the message
				if m.Content.ID == user.String() {
					// return fmt.Errorf("peer %q is the same as the host, ignoring join", m.Content.ID)
					return nil
				}
				if p.Peers.Exist(m.Content.ID) {
					return fmt.Errorf("peer %q already exists, ignoring join", m.Content.ID)
				}

				log.Println("Joining peer", m.Content.ID, "with IP", ip)

				// Create a fake endpoint that could be listened and redirect the packets
				guest, err := client.NewGuestProxy(ip.To4().String())
				if err != nil {
					return fmt.Errorf("could not create guest proxy: %v", err)
				}

				// Add the peer to the list of peers, and start the WebRTC connection
				peer := p.addPeer(m.Content, room, user, guest, true)

				// Create the data channels over the WebRTC connection
				if err := p.createChannels(ctx, peer, guest, room); err != nil {
					return fmt.Errorf("could not create data channels: %v", err)
				}
				return nil
			})
		case signalserver.Leave:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[any]) error {
				peer, ok := p.Peers.Get(m.From)
				if !ok {
					return fmt.Errorf("could not find peer %q", m.From)
				}
				if peer.ID == user.String() {
					return fmt.Errorf("peer %q is the same as the host, ignoring leave", m.From)
				}

				slog.Info("User left", "peer", peer.Name)
				p.Peers.Delete(peer.ID)
				return nil
			})
		case signalserver.RTCOffer:
			return decodeAndRun(buf[1:], func(m signalserver.MessageContent[signalserver.Offer]) error {
				peer := p.addPeer(signalserver.Member{ID: m.From, Name: m.Content.Name}, room, user, nil, false)

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
					From:    user.String(),
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

func (p *PeerToPeer) addPeer(member signalserver.Member, room GameRoom, user User, guest *client.GuestProxy, isJoinNotRTCOffer bool) *client.Peer {
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
		Proxy:      guest,
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
			From:    user.String(),
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
			From: user.String(),
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

			ctx, cancel := context.WithCancel(context.TODO())
			pipe := client.NewPipe(dc)

			switch dc.Label() {
			case fmt.Sprintf("%s/tcp", room):
				go func() {
					if err := guest.RunUDP(ctx, pipe); err != nil {
						panic(err)
					}
				}()
			case fmt.Sprintf("%s/udp", room):
				go func() {
					if err := guest.RunUDP(ctx, pipe); err != nil {
						panic(err)
					}
				}()
			}

			dc.OnClose(func() {
				pipe.Close()
				cancel()
			})
		})
	})

	return peer
}

func (p *PeerToPeer) createChannels(ctx context.Context, peer *client.Peer, guest *client.GuestProxy, room GameRoom) error {
	// UDP
	dcUDP, err := peer.Connection.CreateDataChannel(fmt.Sprintf("%s/udp", room), nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", room, err)
	}
	pipeUDP := client.NewPipe(dcUDP)

	dcUDP.OnError(func(err error) {
		slog.Warn("Data channel error", "error", err)
	})
	dcUDP.OnOpen(func() {
		go func() {
			if err := guest.RunUDP(ctx, pipeUDP); err != nil {
				panic(err)
			}
		}()
	})
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
	pipeTCP := client.NewPipe(dcTCP)

	dcTCP.OnError(func(err error) {
		slog.Warn("Data channel error", "error", err)
	})
	dcTCP.OnOpen(func() {
		go func() {
			if err := guest.RunTCP(ctx, pipeTCP); err != nil {
				panic(err)
			}
		}()
	})
	dcTCP.OnClose(func() {
		log.Printf("dataChannel for %s has closed", peer.ID)
		p.Peers.Delete(peer.ID)
		pipeTCP.Close()
	})

	return nil
}

func (p *PeerToPeer) TodoBroadcast(line []byte) {
	p.Peers.Range(func(s string, peer *client.Peer) {
		peer.ChannelUDP.SendText(string(line))
	})
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

type IpRing struct {
	*ring.Ring
}

func NewIpRing() *IpRing {
	r := ring.New(3)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return &IpRing{r}
}

func (r *IpRing) IP() net.IP {
	d := byte(r.Value.(int))
	defer r.Next()
	return net.IPv4(127, 0, 1, d)
}
