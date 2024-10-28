package internal

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/proxy/proxytesthelper"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

func TestWebRTC(t *testing.T) {
	signalingURL := proxytesthelper.StartSignalServer(t)

	// Player 1
	{
		player1, err := Dial(&DialParams{
			SignalingURL: signalingURL,
			RoomName:     "test",
			ID:           uuid.New().String()[:6],
		})
		if err != nil {
			panic(err)
		}
		go player1.Run(func(peer *Peer, packet webrtc.DataChannelMessage) {
			fmt.Println("Received:", string(packet.Data))
		})
	}

	// Player 2
	{
		player2, err := Dial(&DialParams{
			SignalingURL: signalingURL,
			RoomName:     "test",
			ID:           uuid.New().String()[:6],
		})
		if err != nil {
			panic(err)
		}
		go player2.Run(func(peer *Peer, packet webrtc.DataChannelMessage) {
			fmt.Println("Received:", string(packet.Data))
		})
	}

	<-time.After(3 * time.Second)
}

type FakeSocket struct {
	buf *bytes.Buffer
}

func (fs *FakeSocket) Close() error {
	fs.buf.Reset()
	return nil
}

func (fs *FakeSocket) Read(p []byte) (n int, err error) {
	return fs.buf.Read(p)
}

func (fs *FakeSocket) Write(p []byte) (n int, err error) {
	return fs.buf.Write(p)
}

func TestWebRTCMock(t *testing.T) {
	onMessage := func(peer *Peer, packet webrtc.DataChannelMessage) {
		fmt.Println("Received:", string(packet.Data))
	}

	player1 := &Client{
		ID:    uuid.New().String()[:6],
		Peers: NewPeers(),
		// ws:    &FakeSocket{bytes.NewBuffer([]byte{})},
	}
	player2 := &Client{
		ID:    uuid.New().String()[:6],
		Peers: NewPeers(),
		// ws:    &FakeSocket{bytes.NewBuffer([]byte{})},
	}

	player1.handleJoin(wire.MessageContent[wire.Member]{
		From:    player2.ID,
		Type:    wire.JoinLobby,
		Content: wire.Member{UserID: player2.ID},
		To:      player1.ID,
	}, onMessage)

	<-time.After(3 * time.Second)
}

func TestWebRTCOffer(t *testing.T) {
	myID := uuid.New().String()[:6]

	newMessage := func(msgType wire.EventType, content any) *wire.Message {
		return &wire.Message{
			From:    myID,
			Type:    msgType,
			Content: content,
		}
	}

	// player1 := &Client{
	// 	ID:    uuid.New().String()[:6],
	// 	Peers: NewPeers(),
	// 	ws:    &FakeSocket{bytes.NewBuffer([]byte{})},
	// }
	member := wire.Member{UserID: uuid.New().String()[:6]}

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

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", member.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		msg := newMessage(wire.RTCICECandidate, candidate.ToJSON())
		msg.To = member.UserID

		log.Println(msg)
		// if err := c.SendSignal(msg.ToJSON()); err != nil {
		// 	panic(err)
		// }
	})

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		// if !sendRTCOffer {
		// 	// If this is a message sent first time after joining,
		// 	// then we send the offer to invite yourself to join other users.
		// 	return
		// }

		msg := newMessage(wire.RTCOffer, wire.Offer{
			Member: wire.Member{UserID: myID},
			Offer:  offer,
		})
		msg.To = member.UserID
		log.Println(msg)
		// if err := c.SendSignal(msg.ToJSON()); err != nil {
		// 	panic(err)
		// }
	})

	peerConnection.OnDataChannel(func(channel *webrtc.DataChannel) {
		channel.OnOpen(func() {
			slog.Info("Data channel is open", "channel", channel.Label(), "peer", member.UserID)
			peer.WebRTCDataChannel = channel
		})
	})

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
		slog.Info("Data channel is closed", "peer", member.UserID)
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		// onMessage(peer, msg)
	})

	<-time.After(3 * time.Second)
}
