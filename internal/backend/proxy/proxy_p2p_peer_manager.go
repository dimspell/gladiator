package proxy

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

// PeerToPeerPeerManager manages peer-to-peer connections.
type PeerToPeerPeerManager struct {
	WebRTCConfig webrtc.Configuration
	NewRedirect  redirect.NewRedirect
	Peers        map[*bsession.Session]*PeersToSessionMapping
}

// PeersToSessionMapping maps sessions to their peers.
type PeersToSessionMapping struct {
	IpRing *IpRing
	Game   *GameRoom
	Peers  map[string]*Peer
}

// NewPeerToPeerManager initializes a new PeerToPeerPeerManager.
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
		Peers:        make(map[*bsession.Session]*PeersToSessionMapping),
		NewRedirect:  redirect.New,
	}
}

// setUpChannels sets up the peer connection channels.
func (p *PeerToPeerPeerManager) setUpChannels(session *bsession.Session, player wire.Player, sendRTCOffer bool, createChannels bool) (*Peer, error) {
	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		return nil, err
	}

	peer := p.getOrCreatePeer(session, player)
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

func (p *PeerToPeerPeerManager) getPeer(session *bsession.Session, peerID string) (*Peer, bool) {
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

func (p *PeerToPeerPeerManager) isPeerExisting(session *bsession.Session, player *wire.Player) bool {
	_, ok := p.getPeer(session, player.ID())
	return ok
}

func (p *PeerToPeerPeerManager) setPeer(session *bsession.Session, peer *Peer) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	mapping.Peers[peer.UserID] = peer
}

func (p *PeerToPeerPeerManager) deletePeer(session *bsession.Session, peerID string) {
	mapping, ok := p.Peers[session]
	if !ok {
		return
	}
	delete(mapping.Peers, peerID)
	mapping.Game.DeletePlayer(peerID)
}

func (p *PeerToPeerPeerManager) getOrCreatePeer(session *bsession.Session, player wire.Player) *Peer {
	mapping, ok := p.Peers[session]
	if ok {
		peer, found := mapping.Peers[player.ID()]
		if found {
			return peer
		}
	}

	gameRoom := mapping.Game
	gameRoom.SetPlayer(player)

	isHost := gameRoom.Host.UserID == player.UserID
	isCurrentUser := gameRoom.Host.UserID == session.UserID
	return mapping.IpRing.NextPeerAddress(player.ID(), isCurrentUser, isHost)
}

func (p *PeerToPeerPeerManager) setupPeerConnection(peerConnection *webrtc.PeerConnection, session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
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
		if err := p.handleNegotiation(peerConnection, session, player, sendRTCOffer); err != nil {
			slog.Error("failed to handle negotiation", "error", err)
		}
	})

	return nil
}

func (p *PeerToPeerPeerManager) handleNegotiation(peerConnection *webrtc.PeerConnection, session *bsession.Session, player *wire.Player, sendRTCOffer bool) error {
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	if err := peerConnection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	if sendRTCOffer {
		if err := session.SendRTCOffer(context.TODO(), offer, player.ID()); err != nil {
			return fmt.Errorf("failed to send RTC offer: %w", err)
		}
	}

	return nil
}

func (p *PeerToPeerPeerManager) createDataChannels(peerConnection *webrtc.PeerConnection, session *bsession.Session, peer *Peer) error {
	mapping, found := p.Peers[session]
	if !found {
		return fmt.Errorf("could not find session in peer manager")
	}

	gameRoom := mapping.Game
	roomId := gameRoom.Name

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

	pipe := NewPipe(dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel", "label", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("dataChannel has closed", "label", label)

		pipe.Close()
	})

	return nil
}
