// Package transport defines the transport-agnostic port and dispatch engine
// shared by every proxy mode (relay/QUIC, WebRTC, in-memory test double).
//
// PeerTransport is the hexagon's outer port on the peer-network side. The
// PacketRouter depends only on this interface, so the relay (QUIC), WebRTC, or
// an in-memory test double can be swapped without touching the dispatch logic.
package transport

import (
	"context"

	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
)

// PacketKind classifies the payload of a TransportPacket independent of the
// underlying transport (relay/QUIC, WebRTC, or an in-memory test double).
type PacketKind int

const (
	KindJoin PacketKind = iota
	KindTCP
	KindUDP
	KindLeave
	KindPing
)

// TransportPacket is the transport-agnostic unit delivered between two peers.
// Adapters (e.g. RelayTransport) translate their wire format into this shape so
// the PacketRouter dispatch logic never depends on QUIC or on the RelayPacket
// envelope.
type TransportPacket struct {
	FromID string
	ToID   string
	RoomID string
	Kind   PacketKind
	Data   []byte
}

// PeerTransport is the hexagon's outer port on the peer-network side.
type PeerTransport interface {
	// Join connects to the relay infrastructure and announces presence in roomID.
	Join(ctx context.Context, roomID string) error
	// Send delivers a packet to the peer identified by pkt.ToID.
	Send(ctx context.Context, pkt TransportPacket) error
	// Recv blocks until a packet arrives or ctx is done/cancelled.
	Recv(ctx context.Context) (TransportPacket, error)
	// Leave notifies the infrastructure this peer is departing.
	Leave(ctx context.Context) error
	// Close tears down the transport.
	Close() error
}

// RelayPacket is the wire message exchanged with the relay server. It is defined
// in the shared relay/types package and aliased here so the rest of this package
// can keep using the unqualified name.
type RelayPacket = types.RelayPacket
