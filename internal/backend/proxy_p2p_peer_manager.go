package backend

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

type PeerToPeerPeerManager struct {
	WebRTCConfig webrtc.Configuration
	NewRedirect  redirect.NewRedirect
	Peers        map[*Session]*PeersToSessionMapping
}

type PeersToSessionMapping struct {
	Game  *GameRoom
	Peers map[string]*p2p.Peer
}

func NewPeerToPeerManager() *PeerToPeerPeerManager {
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

	return &PeerToPeerPeerManager{
		WebRTCConfig: config,
		Peers:        make(map[*Session]*PeersToSessionMapping),
		NewRedirect:  redirect.New,
	}
}

func (p *PeerToPeerPeerManager) getPeer(session *Session, peerID string) (*p2p.Peer, bool) {
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

func (p *PeerToPeerPeerManager) isPeerExisting(session *Session, player *wire.Player) bool {
	_, ok := p.getPeer(session, player.ID())
	return ok
}

func (p *PeerToPeerPeerManager) setPeer(session *Session, peer *p2p.Peer) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	mapping.Peers[peer.PeerUserID] = peer
}

func (p *PeerToPeerPeerManager) deletePeer(session *Session, peerID string) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	delete(mapping.Peers, peerID)
}

func (p *PeerToPeerPeerManager) setUpChannels(session *Session, playerId int64, sendRTCOffer bool, createChannels bool) (*p2p.Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		return nil, err
	}

	gameRoom := session.State.GameRoom()
	player, found := gameRoom.GetPlayer(strconv.FormatInt(playerId, 10))
	if !found {
		return nil, fmt.Errorf("could not find player in game room")
	}

	peer := p.getOrCreatePeer(session, &player)
	peer.Connection = peerConnection

	if !p.isPeerExisting(session, &player) {
		p.setPeer(session, peer)
	}

	if err := p.setupPeerConnection(peerConnection, session, &player, sendRTCOffer); err != nil {
		return nil, err
	}

	if createChannels {
		if err := p.createDataChannels(peerConnection, session, peer); err != nil {
			return nil, err
		}
	}

	return peer, nil
}

func (p *PeerToPeerPeerManager) getOrCreatePeer(session *Session, player *wire.Player) *p2p.Peer {
	peer, ok := p.getPeer(session, player.ID())
	if !ok {
		isHost := session.State.GameRoom().Host.UserID == player.UserID
		isCurrentUser := session.State.GameRoom().Host.UserID == session.UserID
		return session.IpRing.NextPeerAddress(player.ID(), isCurrentUser, isHost)
	}
	return peer
}

func (p *PeerToPeerPeerManager) setupPeerConnection(peerConnection *webrtc.PeerConnection, session *Session, player *wire.Player, sendRTCOffer bool) error {
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Debug("ICE Connection State has changed", "peer", player.UserID, "state", connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		if err := session.SendRTCICECandidate(context.TODO(), candidate.ToJSON(), player.ID()); err != nil {
			slog.Error("Could not send ICE candidate", "from", session.GetUserID(), "to", player.UserID, "error", err)
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			slog.Error("failed to create offer", "error", err)
			return
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			slog.Error("failed to set local description", "error", err)
			return
		}

		if !sendRTCOffer {
			// If this is a message sent first time after joining,
			// then we send the offer to invite yourself to join other users.
			return
		}

		if err := session.SendRTCOffer(context.TODO(), offer, player.ID()); err != nil {
			panic(err)
		}
	})

	return nil
}

func (p *PeerToPeerPeerManager) createDataChannels(peerConnection *webrtc.PeerConnection, session *Session, peer *p2p.Peer) error {
	roomId := session.State.GameRoom().Name

	if guestTCP, guestUDP, err := p.NewRedirect(peer.Mode, peer.Addr); err == nil {
		if guestTCP != nil {
			if err := p.createDataChannel(peerConnection, fmt.Sprintf("%s/tcp", roomId), guestTCP); err != nil {
				return err
			}
		}

		if guestUDP != nil {
			if err := p.createDataChannel(peerConnection, fmt.Sprintf("%s/udp", roomId), guestUDP); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *PeerToPeerPeerManager) createDataChannel(peerConnection *webrtc.PeerConnection, label string, redir redirect.Redirect) error {
	dc, err := peerConnection.CreateDataChannel(label, nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %v", label, err)
	}

	pipe := p2p.NewPipe(dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel", "label", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("dataChannel has closed", "label", label)

		pipe.Close()
	})

	return nil
}
