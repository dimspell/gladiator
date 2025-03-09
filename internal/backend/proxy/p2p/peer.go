package p2p

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

// Peer represents a connected player in a game room.
type Peer struct {
	// UserID uniquely identifies the peer
	UserID int64

	// Addr contains network addressing information
	Addr *redirect.Addressing

	// Mode defines the operating mode of the peer
	Mode redirect.Mode

	// Connection holds the WebRTC peer connection
	Connection *webrtc.PeerConnection
	Connected  chan struct{}

	PipeTCP *Pipe
	PipeUDP *Pipe
}

// NewPeer initializes a new Peer based on role.
func NewPeer(connection *webrtc.PeerConnection, r *IpRing, userID int64, isCurrentUser, isHost bool) (*Peer, error) {
	peer := &Peer{
		UserID:     userID,
		Connection: connection,
	}

	switch {
	case isCurrentUser && isHost:
		peer.Addr = &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)}
		peer.Mode = redirect.CurrentUserIsHost
	case isHost == true:
		ip, portTCP, portUDP, err := r.NextAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to get next address: %w", err)
		}
		peer.Addr = &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP}
		peer.Mode = redirect.OtherUserIsHost
	case isHost == false:
		ip, _, portUDP, err := r.NextAddr()
		if err != nil {
			return nil, fmt.Errorf("failed to get next address: %w", err)
		}
		peer.Addr = &redirect.Addressing{IP: ip, UDPPort: portUDP}
		peer.Mode = redirect.OtherUserHasJoined
	default:
		peer.Addr = &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)}
		peer.Mode = redirect.OtherUserIsJoining
	}

	return peer, nil
}

// setupPeerConnection initializes WebRTC event handlers.
func (p *Peer) setupPeerConnection(ctx context.Context, session PeerInterface, player wire.Player, sendRTCOffer bool) error {
	slog.Debug("Setting up peer connection", "userID", player.UserID)

	p.Connection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		slog.Debug("ICE connection state changed", "userID", player.UserID, "state", state.String())
	})

	p.Connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			p.Connected <- struct{}{}
		case webrtc.PeerConnectionStateDisconnected:
			slog.Error("Peer connection disconnected", "userID", player.UserID)
			p.Terminate()
		}
	})

	p.Connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		if err := session.SendRTCICECandidate(ctx, candidate.ToJSON(), player.UserID); err != nil {
			slog.Error("Failed to send ICE candidate", "fromID", session.GetUserID(), "toID", player.UserID, "error", err)
		}
	})

	p.Connection.OnNegotiationNeeded(func() {
		if err := p.handleNegotiation(ctx, session, player, sendRTCOffer); err != nil {
			slog.Error("Failed to handle negotiation", "userID", player.UserID, "error", err)
		}
	})

	return nil
}

// handleNegotiation creates and sends an RTC offer if needed.
func (p *Peer) handleNegotiation(ctx context.Context, session PeerInterface, player wire.Player, sendRTCOffer bool) error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer for peer %d: %w", player.UserID, err)
	}

	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description for peer %d: %w", player.UserID, err)
	}

	if sendRTCOffer {
		if err := session.SendRTCOffer(ctx, offer, player.UserID); err != nil {
			return fmt.Errorf("failed to send RTC offer to peer %d: %w", player.UserID, err)
		}
	}

	return nil
}

// createDataChannels initializes WebRTC data channels for TCP and UDP.
func (p *Peer) createDataChannels(ctx context.Context, newTCPRedirect, newUDPRedirect redirect.NewRedirect, myUserID int64) error {
	if err := p.initDataChannel(ctx, "tcp", myUserID, newTCPRedirect); err != nil {
		return err
	}
	if err := p.initDataChannel(ctx, "udp", myUserID, newUDPRedirect); err != nil {
		return err
	}
	return nil
}

// initDataChannel helps in setting up an individual data channel.
func (p *Peer) initDataChannel(ctx context.Context, proto string, myUserID int64, newRedirect redirect.NewRedirect) error {
	redir, err := newRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create %s redirect: %w", proto, err)
	}

	if redir == nil {
		return nil
	}

	pipe, err := p.createDataChannel(ctx, p.channelName(proto, myUserID, p.UserID), redir)
	if err != nil {
		return fmt.Errorf("failed to create %s channel: %w", proto, err)
	}

	if proto == "tcp" {
		p.PipeTCP = pipe
	} else {
		p.PipeUDP = pipe
	}

	return nil
}

// channelName generates a formatted channel label.
func (p *Peer) channelName(proto string, from, to int64) string {
	return fmt.Sprintf("/redirect/proto/%s/user/%d/to/%d", proto, from, to)
}

// createDataChannel establishes a new WebRTC data channel.
func (p *Peer) createDataChannel(ctx context.Context, label string, redir redirect.Redirect) (*Pipe, error) {
	dc, err := p.Connection.CreateDataChannel(label, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create data channel %q: %w", label, err)
	}

	slog.Debug("Created data channel", "userID", p.UserID, "channel", label)
	pipe := NewPipe(ctx, dc, redir)

	dc.OnOpen(func() {
		slog.Debug("Opened WebRTC channel", "userID", p.UserID, "channel", dc.Label())
	})

	dc.OnClose(func() {
		slog.Info("DataChannel closed", "userID", p.UserID, "channel", label)
		pipe.Close()
	})

	return pipe, nil
}

// Terminate closes all active connections and data channels.
func (p *Peer) Terminate() {
	slog.Debug("Terminating peer connection", "userID", p.UserID)

	if p.Connection != nil {
		if err := p.Connection.Close(); err != nil {
			slog.Error("Failed to close WebRTC connection", "userID", p.UserID, "error", err)
		}
	}

	if p.PipeTCP != nil {
		p.PipeTCP.Close()
	}
	if p.PipeUDP != nil {
		p.PipeUDP.Close()
	}
}

// Pipe manages a WebRTC data channel proxy.
type Pipe struct {
	dc     DataChannel
	done   func()
	proxy  redirect.Redirect
	logger *slog.Logger
}

// DataChannel defines required methods for WebRTC data channels.
type DataChannel interface {
	Label() string
	OnError(func(err error))
	OnMessage(func(msg webrtc.DataChannelMessage))
	OnClose(func())
	Send([]byte) error
	io.Closer
}

// NewPipe creates and starts a new pipe.
func NewPipe(ctx context.Context, dc DataChannel, proxy redirect.Redirect) *Pipe {
	ctx, cancel := context.WithCancel(ctx)
	pipe := &Pipe{
		dc:     dc,
		proxy:  proxy,
		done:   cancel,
		logger: slog.With("label", dc.Label()),
	}

	go func() {
		if err := proxy.Run(ctx, pipe); err != nil {
			pipe.logger.Warn("Proxy failed", "error", err)
			cancel()
		}
	}()

	dc.OnError(func(err error) { pipe.logger.Warn("DataChannel error", "error", err) })
	dc.OnClose(func() {
		pipe.logger.Debug("Closing pipe")
		pipe.Close()
		cancel()
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if _, err := proxy.Write(msg.Data); err != nil {
			pipe.logger.Warn("Failed to write to proxy", "error", err, "data", msg.Data)
		}
	})

	return pipe
}

// Write sends data through the WebRTC data channel.
func (pipe *Pipe) Write(p []byte) (int, error) {
	if err := pipe.dc.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close terminates the pipe.
func (pipe *Pipe) Close() error {
	pipe.done()
	return nil
}
