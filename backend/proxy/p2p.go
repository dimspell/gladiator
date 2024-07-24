package proxy

import (
	"container/ring"
	"context"
	"net"

	"github.com/dimspell/gladiator/proxy/client"
	"github.com/pion/webrtc/v4"
)

type IpRing struct {
	*ring.Ring
}

func NewIpRing() IpRing {
	r := ring.New(100)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return IpRing{r}
}

func (r *IpRing) IP() net.IP {
	d := byte(r.Value.(int))
	defer r.Next()
	return net.IPv4(127, 0, 1, d)
}

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	ipRing IpRing
}

func NewPeerToPeer() *PeerToPeer {
	return &PeerToPeer{
		ipRing: NewIpRing(),
	}
}

func (p *PeerToPeer) Create(localIPAddress string, hostUser string) (net.IP, error) {
	// Connect to the player who is serving the game

	// TODO implement me
	panic("implement me")
}

// HostGame connects to the game host and redirects the traffic to the P2P
// network. The game host is expected to be running on the same machine.
func (p *PeerToPeer) HostGame(gameRoom GameRoom, user User) error {
	host, err := client.ListenHost("localhost:6114")
	if err != nil {
		return err
	}

	// TODO: missing close

	// TODO: Synchronize the closing of the connection
	ctx := context.TODO()

	// Establish connection to the signaling server
	p2p, err := client.Dial(&client.DialParams{
		SignalingURL: "ws://localhost:5050",
		RoomName:     gameRoom.String(),
		ID:           user.String(),
		Name:         user.String(),
	})
	if err != nil {
		panic(err)
	}

	onPeerTCPMessage := func(peer *client.Peer, msg webrtc.DataChannelMessage) {
		// WebRTC => tcp:6114
		host.WriteTCPMessage(ctx, msg.Data)
	}
	onPeerUDPMessage := func(peer *client.Peer, msg webrtc.DataChannelMessage) {
		// WebRTC => udp:6113
		host.WriteUDPMessage(ctx, msg.Data)
	}

	host.OnTCPMessage(ctx, func(msg []byte) {
		// tcp:6114 => WebRTC
		p2p.BroadcastTCP(msg)
	})
	host.OnUDPMessage(ctx, func(msg []byte) {
		// udp:6113 => WebRTC
		p2p.BroadcastUDP(msg)
	})

	go p2p.Run(onPeerUDPMessage, onPeerTCPMessage)

	return nil
}

func (p *PeerToPeer) Join(gameId string, hostUser string, currentPlayer string, ipAddress string) (net.IP, error) {
	ctx := context.TODO()

	ip := p.ipRing.IP()

	guest := client.NewGuestProxy(ipAddress)

	// Establish connection to the signaling server
	p2p, err := client.Dial(&client.DialParams{
		SignalingURL: "ws://localhost:5050",
		RoomName:     gameId,
		ID:           currentPlayer,
		Name:         currentPlayer,
	})
	if err != nil {
		panic(err)
	}

	guest.OnUDPMessage(ctx, func(msg []byte) {
		// udp:6113 => WebRTC
		p2p.BroadcastUDP(msg)
	})
	guest.OnTCPMessage(ctx, func(msg []byte) {
		// tcp:6114 => WebRTC
		p2p.BroadcastTCP(msg)
	})

	onPeerTCPMessage := func(peer *client.Peer, msg webrtc.DataChannelMessage) {
		// WebRTC => tcp[guest]:6114
		guest.WriteTCPMessage(ctx, msg.Data)
	}
	onPeerUDPMessage := func(peer *client.Peer, msg webrtc.DataChannelMessage) {
		// WebRTC => udp:6113
		guest.WriteUDPMessage(ctx, msg.Data)
	}

	go p2p.Run(onPeerUDPMessage, onPeerTCPMessage)
	go guest.Start(ctx)

	return ip, nil
}

func (p *PeerToPeer) Exchange(gameId string, userId string, ipAddress string) (net.IP, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PeerToPeer) GetHostIP(hostIpAddress string) net.IP {
	// TODO implement me
	panic("implement me")
}

func (p *PeerToPeer) Close() {
	// TODO implement me
	panic("implement me")
}
