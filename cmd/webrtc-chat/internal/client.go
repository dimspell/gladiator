package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/url"

	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"golang.org/x/net/websocket"
)

type DialParams struct {
	SignalingURL string
	RoomName     string
	ID           string
}

func Dial(params *DialParams) (*Client, error) {
	// Parse the signaling URL provided from the parameters (command flags)
	u, err := url.Parse(params.SignalingURL)
	if err != nil {
		return nil, err
	}

	// Set parameters
	v := u.Query()
	v.Set("userID", params.ID)
	v.Set("roomName", params.RoomName)
	u.RawQuery = v.Encode()

	slog.Debug("Connecting to the signaling server", "url", u.String())
	ws, err := websocket.Dial(u.String(), "", "http://localhost:8080")
	if err != nil {
		return nil, err
	}

	// Send "hello" message to the signaling server
	req := &wire.Message{
		Type: wire.Hello,
		From: params.ID,
		Content: wire.Member{
			UserID: params.ID,
		},
	}
	if _, err := ws.Write(req.Encode()); err != nil {
		return nil, err
	}

	// Read the response from the signaling server
	buf := make([]byte, 128)
	n, err := ws.Read(buf)
	if err != nil {
		slog.Error("Error reading message", "error", err)
		return nil, err
	}
	if n == 0 || buf[0] != byte(wire.LobbyUsers) {
		return nil, fmt.Errorf("unexpected handshake response: %v", buf[:n])
	}
	resp, err := decodeJSON[wire.MessageContent[string]](buf[1:n])
	if err != nil {
		return nil, err
	}

	slog.Info("Connected to signaling server", "response", resp.Content)

	return &Client{
		ID: params.ID,

		Peers: NewPeers(),
		ws:    ws,
	}, nil
}

func (c *Client) Close() {
	if c == nil {
		return
	}
	if c.ws != nil {
		if err := c.ws.Close(); err != nil {
			slog.Warn("Could not close websocket connection", "error", err)
		}
	}
}

type Client struct {
	ID string

	// ws is the websocket connection to the signaling server
	ws *websocket.Conn

	// peers stores the WebRTC peer connections
	Peers *Peers
}

type MessageHandler func(peer *Peer, packet webrtc.DataChannelMessage)

func (c *Client) Run(onUDP MessageHandler) {
	for {
		buf := make([]byte, 1024)
		n, err := c.ws.Read(buf)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			return
		}

		switch wire.EventType(buf[0]) {
		case wire.Join:
			slog.Warn("JOIN")
			msg, err := decodeJSON[wire.MessageContent[wire.Member]](buf[1:n])
			if err != nil {
				log.Println(err)
				continue
			}
			c.handleJoin(msg, onUDP)
			break
		case wire.Leave:
			slog.Warn("LEAVE")
			msg, err := decodeJSON[wire.MessageContent[any]](buf[1:n])
			if err != nil {
				log.Println(err)
				continue
			}
			c.handleLeave(msg)
			break
		case wire.RTCOffer:
			slog.Warn("RTC_OFFER")
			msg, err := decodeJSON[wire.MessageContent[wire.Offer]](buf[1:n])
			if err != nil {
				log.Println(err)
				continue
			}
			c.handleRTCOffer(msg, onUDP)
			break
		case wire.RTCAnswer:
			slog.Warn("RTC_ANSWER")
			msg, err := decodeJSON[wire.MessageContent[wire.Offer]](buf[1:n])
			if err != nil {
				log.Println(err)
				continue
			}
			c.handleRTCAnswer(msg)
			break
		case wire.RTCICECandidate:
			slog.Warn("RTC_ICE_CANDIDATE")
			msg, err := decodeJSON[wire.MessageContent[webrtc.ICECandidateInit]](buf[1:n])
			if err != nil {
				log.Println(err)
				continue
			}
			c.handleICECandidate(msg)
			break
		default:
			// Do nothing
		}
	}
}

func (c *Client) handleJoin(msg wire.MessageContent[wire.Member], onUDP MessageHandler) {
	slog.Info("Handling join message",
		"id", msg.Content.UserID)

	if msg.Content.UserID == c.ID {
		return
	}

	_, exist := c.Peers.Get(msg.Content.UserID)
	if exist {
		slog.Warn("Member already exists", "id", msg.Content.UserID)
		return
	}

	peer := c.addPeer(msg.Content, true, onUDP)
	c.addNewDataChannel(peer, onUDP)
}

func (c *Client) addPeer(member wire.Member, sendRTCOffer bool, onTCP MessageHandler) *Peer {
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

	peer := &Peer{
		ID:         member.UserID,
		Connection: peerConnection,
	}
	c.Peers.Set(member.UserID, peer)

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", member.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		msg := c.newMessage(wire.RTCICECandidate, candidate.ToJSON())
		msg.To = member.UserID

		if err := c.SendSignal(msg.Encode()); err != nil {
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

		msg := c.newMessage(wire.RTCOffer, wire.Offer{
			Member: wire.Member{UserID: c.ID},
			Offer:  offer,
		})
		msg.To = member.UserID
		if err := c.SendSignal(msg.Encode()); err != nil {
			panic(err)
		}
	})

	peerConnection.OnDataChannel(func(channel *webrtc.DataChannel) {
		channel.OnOpen(func() {
			slog.Info("Data channel is open", "channel", channel.Label(), "peer", member.UserID)
			peer.WebRTCDataChannel = channel
		})
		channel.OnMessage(func(msg webrtc.DataChannelMessage) {
			onTCP(peer, msg)
		})
	})

	return peer
}

func (c *Client) addNewDataChannel(peer *Peer, onMessage MessageHandler) {
	log.Println("Adding new data channel for", peer.ID)

	dc, err := peer.Connection.CreateDataChannel("tcp", nil)
	if err != nil {
		panic(err)
	}
	peer.WebRTCDataChannel = dc

	dc.OnError(func(err error) {
		slog.Warn("Data channel error", "error", err)
	})

	dc.OnOpen(func() {
		peer.WebRTCDataChannel = dc
	})

	dc.OnClose(func() {
		log.Printf("dataChannel for %s has closed", peer.ID)
		c.Peers.Delete(peer.ID)
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		onMessage(peer, msg)
	})
}

func (c *Client) handleLeave(msg wire.MessageContent[any]) {
	peer, ok := c.Peers.Get(msg.From)
	if !ok {
		slog.Error("Could not find peer")
		return
	}

	if peer.ID == c.ID {
		return
	}

	log.Printf("User %s left", peer.Name)
	c.Peers.Delete(peer.ID)
}

func (c *Client) handleRTCOffer(msg wire.MessageContent[wire.Offer], onMessage MessageHandler) {
	peer := c.addPeer(wire.Member{UserID: msg.From}, false, onMessage)

	if err := peer.Connection.SetRemoteDescription(msg.Content.Offer); err != nil {
		panic(err)
	}

	answer, err := peer.Connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err := peer.Connection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	response := c.newMessage(wire.RTCAnswer, wire.Offer{Member: wire.Member{UserID: c.ID}, Offer: answer})
	response.To = msg.From

	if err := c.SendSignal(response.Encode()); err != nil {
		panic(err)
	}
}

func (c *Client) handleRTCAnswer(msg wire.MessageContent[wire.Offer]) {
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.Content.Offer.SDP,
	}

	peer, ok := c.Peers.Get(msg.From)
	if !ok {
		panic("Could not find peer")
	}

	if err := peer.Connection.SetRemoteDescription(answer); err != nil {
		panic(err)
	}
}

func (c *Client) handleICECandidate(msg wire.MessageContent[webrtc.ICECandidateInit]) {
	var candidate = msg.Content

	peer, ok := c.Peers.Get(msg.From)
	if !ok {
		return
	}

	if err := peer.Connection.AddICECandidate(candidate); err != nil {
		panic(err)
	}
}

// newMessage creates a new Message instance
func (c *Client) newMessage(msgType wire.EventType, content any) *wire.Message {
	return &wire.Message{
		From:    c.ID,
		Type:    msgType,
		Content: content,
	}
}

func (c *Client) Broadcast(payload []byte) {
	c.Peers.Range(func(id string, peer *Peer) {
		slog.Debug("Broadcasting message", "to", peer.Name)

		if peer.WebRTCDataChannel == nil {
			slog.Warn("No data channel", "to", peer.Name)
			return
		}

		err := peer.WebRTCDataChannel.Send(payload)
		if err != nil {
			slog.Warn("Error broadcasting message", "to", peer.Name, "error", err)
		}
		return
	})
}

func (c *Client) SendSignal(message []byte) (err error) {
	slog.Info("Sending signal", "payload", string(message))
	_, err = c.ws.Write(message)
	return
}

func decodeJSON[T any](data []byte) (v T, err error) {
	err = json.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding JSON", "error", err, "payload", string(data))
	}
	return
}
