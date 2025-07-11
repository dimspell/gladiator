package p2p

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.SetDiscardLogger()
}

type mockSession struct {
	ID int64

	onSendRTCICECandidate func(webrtc.ICECandidateInit, int64)
	onSendRTCOffer        func(wire.Offer)
	onSendRTCAnswer       func(wire.Offer)
}

func (m mockSession) SendRTCICECandidate(_ context.Context, candidate webrtc.ICECandidateInit, recipientId int64) error {
	if m.onSendRTCICECandidate != nil {
		m.onSendRTCICECandidate(candidate, recipientId)
	}
	return nil
}

func (m mockSession) SendRTCOffer(_ context.Context, sdpOffer webrtc.SessionDescription, recipientId int64) error {
	if m.onSendRTCOffer != nil {
		m.onSendRTCOffer(wire.Offer{
			CreatorID:   m.ID,
			RecipientID: recipientId,
			Offer:       sdpOffer,
		})
	}
	return nil
}

func (m mockSession) SendRTCAnswer(_ context.Context, sdpAnswer webrtc.SessionDescription, recipientId int64) error {
	if m.onSendRTCAnswer != nil {
		m.onSendRTCAnswer(wire.Offer{
			CreatorID:   m.ID,
			RecipientID: recipientId,
			Offer:       sdpAnswer,
		})
	}
	return nil
}

type mockPeerManager struct {
	host  *Peer
	peers map[int64]*Peer
}

func (m *mockPeerManager) AddPeer(peer *Peer) {
	m.peers[peer.UserID] = peer
}

func (m *mockPeerManager) GetPeer(peerId int64) (*Peer, bool) {
	p, ok := m.peers[peerId]
	return p, ok
}

func (m *mockPeerManager) RemovePeer(peerId int64) {
	delete(m.peers, peerId)
}

func (m *mockPeerManager) CreatePeer(player wire.Player) (*Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}

	var mode redirect.Mode
	_, exist := m.peers[player.UserID]
	if !exist {
		mode = redirect.OtherUserIsJoining
	} else {
		mode = redirect.OtherUserHasJoined
	}

	return &Peer{
		UserID:     player.UserID,
		Addr:       nil,
		Mode:       mode,
		Connection: peerConnection,
		Connected:  make(chan struct{}),
		// PipeTCP:    nil,
		// PipeUDP:    nil,
	}, nil
}

func (m *mockPeerManager) Host() (*Peer, bool) {
	return m.host, true
}

func (m *mockPeerManager) SetHost(host *Peer, newHost wire.Player) {
	m.host = host
}

func waitToReceive[T any](ch chan T) T {
	select {
	case res := <-ch: // Wait until offer is received before handling ICE
		return res
	case <-time.After(time.Second * 3):
		break
	}
	var zero T
	return zero
}

func TestPeerToPeerMessageHandler_Handle(t *testing.T) {
	logger.SetPlainTextLogger(os.Stdout, slog.LevelDebug)

	// t.Run("The first player is joining the solo host", func(t *testing.T) {
	// 	ctx := t.Context()
	//
	// 	// Channels for synchronization
	// 	chanOffer := make(chan wire.Offer, 1)
	// 	chanAnswer := make(chan wire.Offer, 1)
	//
	// 	// For player1 (host)
	// 	host := &Peer{UserID: 1, Mode: redirect.CurrentUserIsHost, Connected: make(chan struct{})}
	// 	hostManager := &mockPeerManager{
	// 		host:  host,
	// 		peers: map[int64]*Peer{},
	// 	}
	// 	hostSession := &mockSession{ID: 1,
	// 		onSendRTCOffer: func(offer wire.Offer) {
	// 			chanOffer <- offer
	// 			close(chanOffer) // Close channel after sending offer
	// 		},
	// 	}
	// 	hostHandler := &PeerToPeerMessageHandler{
	// 		UserID:         1,
	// 		peerManager:    hostManager,
	// 		newTCPRedirect: redirect.NewNoop,
	// 		newUDPRedirect: redirect.NewNoop,
	// 		session:        hostSession,
	// 		logger:         slog.With("user", "host"),
	// 	}
	//
	// 	// For player 2 (guest)
	// 	guestManager := &mockPeerManager{}
	// 	first, _ := guestManager.CreatePeer(wire.Player{UserID: 1, Username: "host"})
	// 	second, _ := guestManager.CreatePeer(wire.Player{UserID: 2, Username: "guest"})
	// 	guestManager.host = first
	// 	guestManager.peers = map[int64]*Peer{
	// 		1: first,
	// 		2: second,
	// 	}
	//
	// 	guestSession := &mockSession{
	// 		ID: 2,
	// 		onSendRTCAnswer: func(offer wire.Offer) {
	// 			chanAnswer <- offer
	// 			close(chanAnswer) // Close channel after sending an answer
	// 		},
	// 	}
	// 	guestHandler := &PeerToPeerMessageHandler{
	// 		UserID:         2,
	// 		peerManager:    guestManager,
	// 		newTCPRedirect: redirect.NewNoop,
	// 		newUDPRedirect: redirect.NewNoop,
	// 		session:        guestSession,
	// 		logger:         slog.With("user", "guest"),
	// 	}
	//
	// 	hostSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, recipientID int64) {
	// 		if err := guestHandler.handleRTCCandidate(ctx, ice, hostSession.ID); err != nil {
	// 			log.Printf("AddICECandidate returned error: %v", err)
	// 		}
	// 	}
	// 	guestSession.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, recipientID int64) {
	// 		if err := hostHandler.handleRTCCandidate(ctx, ice, guestSession.ID); err != nil {
	// 			log.Printf("handleRTCCandidate returned error: %v", err)
	// 		}
	// 	}
	//
	// 	// New player is joining (host handles join)
	// 	guest := wire.Player{UserID: 2}
	// 	if err := hostHandler.handleJoinRoom(ctx, guest); err != nil {
	// 		t.Errorf("handleJoinRoom returned error: %v", err)
	// 		return
	// 	}
	//
	// 	// Send RTC Offer and handle it (guest handles RTC Offer)
	// 	offer := waitToReceive(chanOffer)
	// 	if err := guestHandler.handleRTCOffer(ctx, offer, host.UserID); err != nil {
	// 		log.Printf("handleRTCOffer: %v", err)
	// 		return
	// 	}
	//
	// 	// Send RTC Answer and handle it (host handles RTC Answer)
	// 	answer := waitToReceive(chanAnswer)
	// 	if err := hostHandler.handleRTCAnswer(ctx, answer, guest.UserID); err != nil {
	// 		log.Printf("handleRTCAnswer: %v", err)
	// 		return
	// 	}
	//
	// 	select {
	// 	case <-second.Connected:
	// 		t.Logf("guest established connection with the host")
	// 		return
	// 	case <-time.After(time.Second * 3):
	// 		t.Errorf("timed out waiting to connect")
	// 		return
	// 	}
	//
	// 	// var wg sync.WaitGroup
	// 	// wg.Add(1)
	// 	// //
	// 	// second.Connection.OnDataChannel(func(dc *webrtc.DataChannel) {
	// 	// 	t.Logf("got data channel %q", dc.Label())
	// 	// 	//
	// 	// 	wg.Done()
	// 	// })
	// 	// wg.Wait()
	// })

	// t.Run("Two players are joining the solo host and interconnect", func(t *testing.T) {
	// 	ctx := t.Context()
	//
	// 	// Channels for synchronization
	// 	offerHostToGuest2Chan := make(chan wire.Offer, 1)
	// 	answerGuest2ToHostChan := make(chan wire.Offer, 1)
	// 	offerHostToGuest3Chan := make(chan wire.Offer, 1)
	// 	answerGuest3ToHostChan := make(chan wire.Offer, 1)
	// 	offerGuest2ToGuest3Chan := make(chan wire.Offer, 1)
	// 	answerGuest3ToGuest2Chan := make(chan wire.Offer, 1)
	//
	// 	// Host (Player 1) setup
	// 	hostPlayer1 := &Peer{UserID: 1, Connected: make(chan struct{})}
	// 	hostPlayer1Manager := &mockPeerManager{
	// 		host:  hostPlayer1,
	// 		peers: map[int64]*Peer{},
	// 	}
	// 	hostPlayer1Session := &mockSession{ID: 1}
	// 	hostPlayer1Handler := &PeerToPeerMessageHandler{
	// 		UserID:         1,
	// 		peerManager:    hostPlayer1Manager,
	// 		newTCPRedirect: redirect.NewNoop,
	// 		newUDPRedirect: redirect.NewNoop,
	// 		session:        hostPlayer1Session,
	// 		logger:         slog.With("user", "hostPlayer1"),
	// 	}
	//
	// 	// Guest Player 2 setup
	// 	guestPlayer2Manager := &mockPeerManager{}
	// 	guestPlayer2_hostPeer, _ := guestPlayer2Manager.CreatePeer(wire.Player{UserID: 1, Username: "hostPlayer1"})
	// 	guestPlayer2_selfPeer, _ := guestPlayer2Manager.CreatePeer(wire.Player{UserID: 2, Username: "guestPlayer2"})
	// 	guestPlayer2Manager.host = guestPlayer2_hostPeer
	// 	guestPlayer2Manager.peers = map[int64]*Peer{1: guestPlayer2_hostPeer, 2: guestPlayer2_selfPeer}
	// 	guestPlayer2Session := &mockSession{ID: 2}
	// 	guestPlayer2Handler := &PeerToPeerMessageHandler{
	// 		UserID:         2,
	// 		peerManager:    guestPlayer2Manager,
	// 		newTCPRedirect: redirect.NewNoop,
	// 		newUDPRedirect: redirect.NewNoop,
	// 		session:        guestPlayer2Session,
	// 		logger:         slog.With("user", "guestPlayer2"),
	// 	}
	//
	// 	// Guest Player 3 setup
	// 	guestPlayer3Manager := &mockPeerManager{}
	// 	guestPlayer3_hostPeer, _ := guestPlayer3Manager.CreatePeer(wire.Player{UserID: 1, Username: "hostPlayer1"})
	// 	guestPlayer3_otherPeer, _ := guestPlayer3Manager.CreatePeer(wire.Player{UserID: 2, Username: "guestPlayer2"})
	// 	guestPlayer3_selfPeer, _ := guestPlayer3Manager.CreatePeer(wire.Player{UserID: 3, Username: "guestPlayer3"})
	// 	guestPlayer3Manager.host = guestPlayer3_hostPeer
	// 	guestPlayer3Manager.peers = map[int64]*Peer{1: guestPlayer3_hostPeer, 2: guestPlayer3_otherPeer, 3: guestPlayer3_selfPeer}
	// 	guestPlayer3Session := &mockSession{ID: 3}
	// 	guestPlayer3Handler := &PeerToPeerMessageHandler{
	// 		UserID:         3,
	// 		peerManager:    guestPlayer3Manager,
	// 		newTCPRedirect: redirect.NewNoop,
	// 		newUDPRedirect: redirect.NewNoop,
	// 		session:        guestPlayer3Session,
	// 		logger:         slog.With("user", "guestPlayer3"),
	// 	}
	//
	// 	// ICE Candidate Handling
	// 	hostPlayer1Session.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
	// 		switch fromUserID {
	// 		case 2:
	// 			if err := guestPlayer2Handler.handleRTCCandidate(ctx, ice, hostPlayer1Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		case 3:
	// 			if err := guestPlayer3Handler.handleRTCCandidate(ctx, ice, hostPlayer1Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		}
	// 	}
	//
	// 	guestPlayer2Session.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
	// 		switch fromUserID {
	// 		case 1:
	// 			if err := hostPlayer1Handler.handleRTCCandidate(ctx, ice, guestPlayer2Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		case 3:
	// 			if err := guestPlayer3Handler.handleRTCCandidate(ctx, ice, guestPlayer2Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		}
	// 	}
	//
	// 	guestPlayer3Session.onSendRTCICECandidate = func(ice webrtc.ICECandidateInit, fromUserID int64) {
	// 		switch fromUserID {
	// 		case 1:
	// 			if err := hostPlayer1Handler.handleRTCCandidate(ctx, ice, guestPlayer3Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		case 2:
	// 			if err := guestPlayer2Handler.handleRTCCandidate(ctx, ice, guestPlayer3Session.ID); err != nil {
	// 				t.Fatal(err)
	// 			}
	// 		}
	// 	}
	//
	// 	// Offer/Answer Handling
	// 	hostPlayer1Session.onSendRTCOffer = func(offer wire.Offer) {
	// 		t.Log(offer.CreatorID, offer.RecipientID)
	//
	// 		switch offer.CreatorID {
	// 		case 2:
	// 			offerHostToGuest2Chan <- offer
	// 			close(offerHostToGuest2Chan)
	// 		case 3:
	// 			offerHostToGuest3Chan <- offer
	// 			close(offerHostToGuest3Chan)
	// 		default:
	// 			t.Fatal("Unexpected offer")
	// 		}
	// 	}
	//
	// 	guestPlayer2Session.onSendRTCOffer = func(offer wire.Offer) {
	// 		switch {
	// 		case offer.CreatorID == 3:
	// 			offerGuest2ToGuest3Chan <- offer
	// 			close(offerGuest2ToGuest3Chan)
	// 		default:
	// 			t.Fatal("Unexpected offer")
	// 		}
	// 	} // Guest 2 doesn't send offers in this scenario
	//
	// 	guestPlayer2Session.onSendRTCAnswer = func(offer wire.Offer) {
	// 		switch offer.CreatorID {
	// 		case 1:
	// 			answerGuest2ToHostChan <- offer
	// 			close(answerGuest2ToHostChan)
	// 		default:
	// 			t.Fatal("Unexpected offer")
	// 		}
	// 	}
	//
	// 	guestPlayer3Session.onSendRTCAnswer = func(offer wire.Offer) {
	// 		switch {
	// 		case offer.CreatorID == 1:
	// 			answerGuest3ToHostChan <- offer
	// 			close(answerGuest3ToHostChan)
	// 		case offer.CreatorID == 2:
	// 			answerGuest3ToGuest2Chan <- offer
	// 			close(answerGuest3ToGuest2Chan)
	// 		default:
	// 			t.Fatal("Unexpected offer", offer.CreatorID)
	// 		}
	// 	}
	//
	// 	guestPlayer3Session.onSendRTCOffer = func(offer wire.Offer) {
	// 		t.Fatal("Must not send RTC offer")
	// 	}
	//
	// 	// Guest Player 2 joins host
	// 	assert.NoError(t, hostPlayer1Handler.handleJoinRoom(ctx, wire.Player{UserID: 2}))
	// 	offerHostToGuest2 := waitToReceive(offerHostToGuest2Chan)
	// 	assert.NoError(t, guestPlayer2Handler.handleRTCOffer(ctx, offerHostToGuest2, 1))
	// 	answerGuest2ToHost := waitToReceive(answerGuest2ToHostChan)
	// 	assert.NoError(t, hostPlayer1Handler.handleRTCAnswer(ctx, answerGuest2ToHost, 2))
	//
	// 	// // Guest Player 3 joins host
	// 	// assert.NoError(t, hostPlayer1Handler.handleJoinRoom(ctx, wire.Player{UserID: 3}))
	// 	// offerHostToGuest3 := waitToReceive(offerHostToGuest3Chan)
	// 	// assert.NoError(t, guestPlayer3Handler.handleRTCOffer(ctx, offerHostToGuest3, 1))
	// 	// answerGuest3ToHost := waitToReceive(answerGuest3ToHostChan)
	// 	// assert.NoError(t, hostPlayer1Handler.handleRTCAnswer(ctx, answerGuest3ToHost, 3))
	// 	//
	// 	// // Guest Player 3 connects to Guest Player 2
	// 	// assert.NoError(t, guestPlayer2Handler.handleJoinRoom(ctx, wire.Player{UserID: 3}))
	// 	// offerGuest2ToGuest3 := waitToReceive(offerGuest2ToGuest3Chan)
	// 	// assert.NoError(t, guestPlayer3Handler.handleRTCOffer(ctx, offerGuest2ToGuest3, 2))
	// 	// answerGuest3ToGuest2 := waitToReceive(answerGuest3ToGuest2Chan)
	// 	// assert.NoError(t, guestPlayer2Handler.handleRTCAnswer(ctx, answerGuest3ToGuest2, 3))
	//
	// 	waitForConnection := func(t *testing.T, wg *sync.WaitGroup, peer *Peer, channelNames ...string) {
	// 		connectedChan := make(chan struct{}, 1)
	//
	// 		peer.Connection.OnDataChannel(func(dc *webrtc.DataChannel) {
	// 			dc.OnError(func(e error) {
	// 				t.Error(e)
	// 			})
	// 			dc.OnOpen(func() {
	// 				for _, name := range channelNames {
	// 					if dc.Label() == name {
	// 						connectedChan <- struct{}{}
	// 					}
	// 				}
	// 				// connectedChan <- struct{}{}
	// 			})
	// 		})
	//
	// 		select {
	// 		case <-connectedChan:
	// 			wg.Done()
	// 			t.Logf("peer %d established connection with the host", peer.UserID)
	// 		case <-time.After(time.Second * 3):
	// 			wg.Done()
	// 			t.Errorf("timed out waiting to connect %d", peer.UserID)
	// 		}
	// 		close(connectedChan)
	// 	}
	//
	// 	wg := new(sync.WaitGroup)
	// 	wg.Add(1)
	// 	// wg := new(sync.WaitGroup)
	// 	// wg.Add(3)
	// 	//
	// 	// go waitForConnection(t, wg, guestPlayer2_selfPeer, "/redirect/proto/game/user/1/to/2")
	// 	// go waitForConnection(t, wg, guestPlayer3_selfPeer, "/redirect/proto/game/user/1/to/3")
	// 	// go waitForConnection(t, wg, guestPlayer3_otherPeer, "/redirect/proto/game/user/2/to/3")
	// 	//
	// 	// wg.Wait()
	//
	// 	waitForConnection(t, wg, guestPlayer3_selfPeer, "/redirect/proto/game/user/1/to/3")
	// })
}

func TestPeerToPeerMessageHandler_handleLeave(t *testing.T) {
	t.Run("Peer leaves room", func(t *testing.T) {
		peerManager := &mockPeerManager{
			peers: map[int64]*Peer{
				2: {UserID: 2},
			},
		}
		h := &PeerToPeerMessageHandler{
			session:     &mockSession{ID: 1},
			peerManager: peerManager,
			logger:      slog.Default(),
		}

		leavingPlayer := wire.Player{UserID: 2}
		assert.NoError(t, h.handleLeaveRoom(t.Context(), leavingPlayer))
		_, ok := peerManager.peers[leavingPlayer.UserID]
		assert.False(t, ok, "Peer should be removed from peerManager")
	})
}

func TestPeerToPeerMessageHandler_handleHostMigration(t *testing.T) {
	t.Run("I am a host, switching to new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, I will become new host", func(t *testing.T) {})

	t.Run("Host left, I am a guest, other become host", func(t *testing.T) {
		player1 := &Peer{
			UserID:     1, // host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
		}
		player3 := &Peer{
			UserID:     3, // to-be-host
			Addr:       nil,
			Mode:       0,
			Connection: nil,
			Connected:  nil,
		}
		newHostPlayer := wire.Player{UserID: 3}

		peerManager := &mockPeerManager{
			host: player1,
			peers: map[int64]*Peer{
				1: player1,
				3: player3,
			},
		}
		h := &PeerToPeerMessageHandler{
			UserID:         2,
			session:        &mockSession{ID: 2},
			peerManager:    peerManager,
			newTCPRedirect: redirect.NewNoop,
			newUDPRedirect: redirect.NewNoop,
			logger:         slog.Default(),
		}
		if err := h.handleHostMigration(t.Context(), newHostPlayer); err != nil {
			t.Error(err)
		}
		if peerManager.host.UserID != 3 {
			t.Error("host not migrated")
		}
	})
}
