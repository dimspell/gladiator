package p2p

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/transport"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTransport() *webrtcTransport {
	return &webrtcTransport{
		logger: slog.Default(),
		lookup: func(peerID string) (*Peer, bool) { return nil, false },
		recvCh: make(chan transport.TransportPacket, 8),
	}
}

func TestWebRTCTransport_Send_TCPPrefix(t *testing.T) {
	peer := &Peer{peerID: "200", logger: slog.Default()}
	tr := &webrtcTransport{
		logger: slog.Default(),
		lookup: func(id string) (*Peer, bool) { return peer, id == "200" },
		recvCh: make(chan transport.TransportPacket, 8),
	}

	require.NoError(t, tr.Send(context.Background(), transport.TransportPacket{ToID: "200", Kind: transport.KindTCP, Data: []byte("hi")}))

	peer.mu.Lock()
	got := peer.outboundQueue
	peer.mu.Unlock()
	require.Len(t, got, 1)
	assert.Equal(t, byte('T'), got[0][0])
	assert.Equal(t, []byte("hi"), got[0][1:])
}

func TestWebRTCTransport_Send_UDPPrefix(t *testing.T) {
	peer := &Peer{peerID: "200", logger: slog.Default()}
	tr := &webrtcTransport{
		logger: slog.Default(),
		lookup: func(id string) (*Peer, bool) { return peer, id == "200" },
		recvCh: make(chan transport.TransportPacket, 8),
	}

	require.NoError(t, tr.Send(context.Background(), transport.TransportPacket{ToID: "200", Kind: transport.KindUDP, Data: []byte("hi")}))

	peer.mu.Lock()
	got := peer.outboundQueue
	peer.mu.Unlock()
	require.Len(t, got, 1)
	assert.Equal(t, byte('U'), got[0][0])
	assert.Equal(t, []byte("hi"), got[0][1:])
}

func TestWebRTCTransport_Send_UnknownPeer(t *testing.T) {
	tr := newTestTransport()
	// Should not error and should not panic; just drops.
	assert.NoError(t, tr.Send(context.Background(), transport.TransportPacket{ToID: "999", Kind: transport.KindTCP, Data: []byte("x")}))
}

func TestWebRTCTransport_Send_NonDataKindsAreDropped(t *testing.T) {
	peer := &Peer{peerID: "200", logger: slog.Default()}
	tr := &webrtcTransport{
		logger: slog.Default(),
		lookup: func(id string) (*Peer, bool) { return peer, id == "200" },
		recvCh: make(chan transport.TransportPacket, 8),
	}
	for _, kind := range []transport.PacketKind{transport.KindJoin, transport.KindLeave, transport.KindPing} {
		assert.NoError(t, tr.Send(context.Background(), transport.TransportPacket{ToID: "200", Kind: kind, Data: []byte("x")}))
	}
	peer.mu.Lock()
	defer peer.mu.Unlock()
	assert.Empty(t, peer.outboundQueue)
}

func TestWebRTCTransport_Recv_Deliver(t *testing.T) {
	tr := newTestTransport()
	tr.deliver("100", transport.KindUDP, []byte("yo"))

	pkt, err := tr.Recv(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "100", pkt.FromID)
	assert.Equal(t, transport.KindUDP, pkt.Kind)
	assert.Equal(t, []byte("yo"), pkt.Data)
}

func TestWebRTCTransport_Recv_ContextCancel(t *testing.T) {
	tr := newTestTransport()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tr.Recv(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWebRTCTransport_Close_UnblocksRecv(t *testing.T) {
	tr := newTestTransport()
	tr.Close()

	_, err := tr.Recv(context.Background())
	assert.ErrorIs(t, err, io.EOF)
}

func TestWebRTCTransport_Join_RecreatesChannel(t *testing.T) {
	tr := newTestTransport()
	tr.Close() // close the initial channel

	require.NoError(t, tr.Join(context.Background(), "room"))
	tr.deliver("100", transport.KindTCP, []byte("again"))

	pkt, err := tr.Recv(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("again"), pkt.Data)
}

// TestWebRTCTransport_EndToEnd proves the unified data plane forwards real game
// packets over a live WebRTC data channel: A's transport.Send prefixes and writes
// onto the data channel, B's OnMessage delivers it into its transport, and B's
// Recv returns the original payload. This is the link the unit tests don't cover
// (actual byte transfer through pion) and the acceptance suite doesn't assert
// (it checks signaling/room state, not packet forwarding).
func TestWebRTCTransport_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping WebRTC e2e in short mode")
	}

	pcA, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	defer pcA.Close()
	pcB, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	require.NoError(t, err)
	defer pcB.Close()

	// B receives the data channel A creates.
	dcBCh := make(chan *webrtc.DataChannel, 1)
	pcB.OnDataChannel(func(dc *webrtc.DataChannel) {
		dcBCh <- dc
	})

	dcA, err := pcA.CreateDataChannel("game", nil)
	require.NoError(t, err)

	// Exchange ICE candidates over loopback.
	pcA.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			require.NoError(t, pcB.AddICECandidate(c.ToJSON()))
		}
	})
	pcB.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			require.NoError(t, pcA.AddICECandidate(c.ToJSON()))
		}
	})

	offer, err := pcA.CreateOffer(nil)
	require.NoError(t, err)
	require.NoError(t, pcA.SetLocalDescription(offer))
	require.NoError(t, pcB.SetRemoteDescription(offer))
	answer, err := pcB.CreateAnswer(nil)
	require.NoError(t, err)
	require.NoError(t, pcB.SetLocalDescription(answer))
	require.NoError(t, pcA.SetRemoteDescription(answer))

	var dcB *webrtc.DataChannel
	select {
	case dcB = <-dcBCh:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for data channel on B")
	}

	openA := make(chan struct{})
	dcA.OnOpen(func() { close(openA) })
	openB := make(chan struct{})
	dcB.OnOpen(func() { close(openB) })
	select {
	case <-openA:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for data channel A to open")
	}
	select {
	case <-openB:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for data channel B to open")
	}

	// A's peer object for B carries dcA; B's peer object for A carries dcB.
	// Sending to a peer uses the local end of the channel, which delivers to the
	// remote end's OnMessage.
	peerA := &Peer{peerID: "A", connection: pcA, dataChannel: dcA, logger: slog.Default()}
	peerB := &Peer{peerID: "B", connection: pcB, dataChannel: dcB, logger: slog.Default()}

	trA := &webrtcTransport{
		logger: slog.Default(),
		lookup: func(id string) (*Peer, bool) { return peerA, id == "B" },
		recvCh: make(chan transport.TransportPacket, 16),
	}
	trB := &webrtcTransport{
		logger: slog.Default(),
		lookup: func(id string) (*Peer, bool) { return peerB, id == "A" },
		recvCh: make(chan transport.TransportPacket, 16),
	}

	// Wire inbound data-channel messages into the receiving transport.
	dcB.OnMessage(func(msg webrtc.DataChannelMessage) {
		if len(msg.Data) < 2 {
			return
		}
		var kind transport.PacketKind
		switch msg.Data[0] {
		case 'T':
			kind = transport.KindTCP
		case 'U':
			kind = transport.KindUDP
		default:
			return
		}
		trB.deliver("A", kind, msg.Data[1:])
	})
	dcA.OnMessage(func(msg webrtc.DataChannelMessage) {
		if len(msg.Data) < 2 {
			return
		}
		var kind transport.PacketKind
		switch msg.Data[0] {
		case 'T':
			kind = transport.KindTCP
		case 'U':
			kind = transport.KindUDP
		default:
			return
		}
		trA.deliver("B", kind, msg.Data[1:])
	})

	// A -> B (TCP)
	require.NoError(t, trA.Send(context.Background(), transport.TransportPacket{
		ToID: "B", Kind: transport.KindTCP, Data: []byte("hello-world"),
	}))

	pkt, err := trB.Recv(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "A", pkt.FromID)
	assert.Equal(t, transport.KindTCP, pkt.Kind)
	assert.Equal(t, []byte("hello-world"), pkt.Data)

	// B -> A (UDP)
	require.NoError(t, trB.Send(context.Background(), transport.TransportPacket{
		ToID: "A", Kind: transport.KindUDP, Data: []byte("pong"),
	}))

	pkt, err = trA.Recv(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "B", pkt.FromID)
	assert.Equal(t, transport.KindUDP, pkt.Kind)
	assert.Equal(t, []byte("pong"), pkt.Data)
}
