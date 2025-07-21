package p2p

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
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

	// PipeTCP *Pipe
	// PipeUDP *Pipe
	PipeRouter *PipeRouter
}

// NewPeer initializes a new Peer.
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
func (p *Peer) setupPeerConnection(ctx context.Context, logger *slog.Logger, session PeerInterface, playerId int64, sendRTCOffer bool) error {
	logger.Debug("Setting up peer connection")

	p.Connection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		logger.Debug("ICE connection state changed", "state", state.String())
	})

	p.Connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			if p.Connected != nil {
				p.Connected <- struct{}{}
			}
		case webrtc.PeerConnectionStateDisconnected:
			logger.Error("Peer connection disconnected")
			p.Terminate()
		}
	})

	p.Connection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		if err := session.SendRTCICECandidate(ctx, candidate.ToJSON(), playerId); err != nil {
			logger.Error("Failed to send ICE candidate", "fromID", p.UserID, "toID", playerId, logging.Error(err))
		}
	})

	p.Connection.OnNegotiationNeeded(func() {
		slog.With("user", playerId).Debug("Negotiation needed")

		if err := p.handleNegotiation(ctx, session, playerId, sendRTCOffer); err != nil {
			logger.Error("Failed to handle negotiation", "userID", playerId, logging.Error(err))
		}
	})

	return nil
}

// handleNegotiation creates and sends an RTC offer if needed.
func (p *Peer) handleNegotiation(ctx context.Context, session PeerInterface, playerId int64, sendRTCOffer bool) error {
	offer, err := p.Connection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer for peer %d: %w", playerId, err)
	}

	if err := p.Connection.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description for peer %d: %w", playerId, err)
	}

	if sendRTCOffer {
		slog.Info("Sending RTC offer to peer", "playerId", playerId, "peerUserId", p.UserID)

		if err := session.SendRTCOffer(ctx, offer, playerId); err != nil {
			return fmt.Errorf("failed to send RTC offer to peer %d: %w", playerId, err)
		}
	}

	return nil
}

// createDataChannels initializes WebRTC data channels for TCP and UDP.
func (p *Peer) createDataChannels(ctx context.Context, logger *slog.Logger, newTCPRedirect, newUDPRedirect redirect.NewRedirect, myUserID int64) error {
	redirTCP, err := newTCPRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create TCP redirect: %w", err)
	}
	redirUDP, err := newUDPRedirect(p.Mode, p.Addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP redirect: %w", err)
	}

	label := p.channelName("game", myUserID, p.UserID)

	dc, err := p.Connection.CreateDataChannel(label, nil)
	if err != nil {
		return fmt.Errorf("could not create data channel %q: %w", label, err)
	}

	logger = logger.With("channel_id", label)
	logger.Debug("Created data channel")

	p.PipeRouter = NewPipeRouter(ctx, logger, dc, redirTCP, redirUDP)

	// if err := p.initDataChannel(ctx, logger, "tcp", myUserID, newTCPRedirect); err != nil {
	// 	return err
	// }
	// if err := p.initDataChannel(ctx, logger, "udp", myUserID, newUDPRedirect); err != nil {
	// 	return err
	// }
	return nil
}

// channelName generates a formatted channel label.
func (p *Peer) channelName(proto string, from, to int64) string {
	return fmt.Sprintf("/redirect/proto/%s/user/%d/to/%d", proto, from, to)
}

// Terminate closes all active connections and data channels.
func (p *Peer) Terminate() {
	slog.Debug("Terminating peer connection", "userID", p.UserID)

	if p.Connection != nil {
		if err := p.Connection.GracefulClose(); err != nil {
			slog.Error("Failed to close WebRTC connection", "userID", p.UserID, logging.Error(err))
		}
	}

	if p.PipeRouter != nil {
		if err := p.PipeRouter.Close(); err != nil {
			slog.Error("Failed to close the game pipe router", "userID", p.UserID, logging.Error(err))
		}
	}

	// if p.PipeTCP != nil {
	// 	if err := p.PipeTCP.Close(); err != nil {
	// 		slog.Error("Failed to close TCP pipe", "userID", p.CreatorID, logging.Error(err))
	// 	}
	// }
	// if p.PipeUDP != nil {
	// 	if err := p.PipeUDP.Close(); err != nil {
	// 		slog.Error("Failed to close UDP pipe", "userID", p.CreatorID, logging.Error(err))
	// 	}
	// }
}

type PipeRouter struct {
	dc     DataChannel
	done   func()
	logger *slog.Logger

	proxyTCP redirect.Redirect
	proxyUDP redirect.Redirect
}

func NewPipeRouter(ctx context.Context, logger *slog.Logger, dc DataChannel, tcpProxy, udpProxy redirect.Redirect) *PipeRouter {
	ctx, cancel := context.WithCancel(ctx)
	pipe := &PipeRouter{
		dc:       dc,
		proxyTCP: tcpProxy,
		proxyUDP: udpProxy,
		done:     cancel,
		logger:   logger,
	}

	g, gctx := errgroup.WithContext(ctx)

	if tcpProxy != nil {
		// tcpProxy.OnReceive = func(p []byte) error {
		// 	_, err := pipe.WriteTCP(p)
		// 	return err
		// }

		g.Go(func() error {
			return tcpProxy.Run(gctx)
		})
	}
	if udpProxy != nil {
		// udpProxy.OnReceive = func(p []byte) error {
		// 	_, err := pipe.WriteUDP(p)
		// 	return err
		// }

		g.Go(func() error {
			return udpProxy.Run(gctx)
		})
	}

	go func() {
		if err := g.Wait(); err != nil {
			pipe.logger.Warn("Proxy failed", logging.Error(err))
			cancel()

			pipe.logger.Warn("Closing data-channel", "error", dc.Close())
		}
	}()

	dc.OnOpen(func() {
		pipe.logger.Debug("Opened WebRTC channel")
	})

	dc.OnError(func(err error) { pipe.logger.Warn("DataChannel error", logging.Error(err)) })
	dc.OnClose(func() {
		pipe.logger.Debug("Closing pipe")
		pipe.Close()
		cancel()
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		switch msg.Data[0] {
		case 'T':
			if _, err := tcpProxy.Write(msg.Data[1:]); err != nil {
				pipe.logger.Warn("Failed to write to proxy", logging.Error(err), "data", msg.Data)
			}
		case 'U':
			if _, err := udpProxy.Write(msg.Data[1:]); err != nil {
				pipe.logger.Warn("Failed to write to proxy", logging.Error(err), "data", msg.Data)
			}
		}
	})

	return pipe
}

func (pipe *PipeRouter) WriteUDP(p []byte) (int, error) {
	return pipe.WriteToChannel(p, 'U')
}

func (pipe *PipeRouter) WriteTCP(p []byte) (int, error) {
	return pipe.WriteToChannel(p, 'T')
}

func (pipe *PipeRouter) WriteToChannel(p []byte, proto byte) (int, error) {
	payload := make([]byte, len(p)+1)
	payload[0] = proto
	copy(payload[1:], p)

	if err := pipe.dc.Send(payload); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close terminates the pipe router.
func (pipe *PipeRouter) Close() error {
	pipe.done()
	return nil
}

// DataChannel defines required methods for WebRTC data channels.
type DataChannel interface {
	io.Closer

	OnOpen(func())
	OnClose(func())
	OnError(func(err error))
	Label() string
	OnMessage(func(msg webrtc.DataChannelMessage))
	Send([]byte) error
}
