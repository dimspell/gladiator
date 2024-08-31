package p2p

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

var api *webrtc.API

func newWebrtcPeer() *webrtc.PeerConnection {
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

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			fmt.Println("Peer Connection has gone to closed exiting")
		}
	})

	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", dc.Label(), dc.ID())

		dc.OnOpen(func() {
			log.Println(dc.Label(), "opened")
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", dc.Label(), string(msg.Data))
		})
	})

	// Todo Register channel creation

	return peerConnection
}

func acceptOfferSendAnswer(peer *webrtc.PeerConnection, offer webrtc.SessionDescription) webrtc.SessionDescription {
	if err := peer.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	// <-webrtc.GatheringCompletePromise(peer)

	answer, err := peer.CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = peer.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	return answer
}

func exchangeIceCandidates(peer1 *webrtc.PeerConnection, peer2 *webrtc.PeerConnection) {
	peer1.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		fmt.Println("Offerer candidate:", candidate.String())
		if err := peer2.AddICECandidate(candidate.ToJSON()); err != nil {
			panic(err)
		}
	})
}

func negotiate(peerConnection *webrtc.PeerConnection, ch chan<- webrtc.SessionDescription) {
	peerConnection.OnNegotiationNeeded(func() {
		offer, err := peerConnection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}

		if err := peerConnection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		log.Println("Offer")
		ch <- offer
	})
}

func TestCreateTwoPeers(t *testing.T) {
	peer1 := newWebrtcPeer()
	peer2 := newWebrtcPeer()

	ctx, done := context.WithTimeout(context.TODO(), 3*time.Second)
	defer done()

	peer1Chan := make(chan webrtc.SessionDescription)
	negotiate(peer1, peer1Chan)

	peer2Chan := make(chan webrtc.SessionDescription)
	negotiate(peer2, peer2Chan)

	go func() {
		// if ctx.Err() == nil {
		// 	done()
		// }
		for {
			select {
			case <-ctx.Done():
				log.Println("Done")
				return
			case offer := <-peer1Chan:
				log.Println("Offer 1:")

				acceptOfferSendAnswer(peer2, offer)

				// exchangeIceCandidates(peer1, peer2)
				// exchangeIceCandidates(peer2, peer1)

				log.Println("Answer 2:")

				continue
			case <-peer2Chan:
				log.Println("Offer 2:")
				continue
			}
		}
	}()

	dcPeer1, err := peer1.CreateDataChannel(roomName, nil)
	if err != nil {
		panic(err)
	}
	defer dcPeer1.Close()

	dcPeer1.OnMessage(func(msg webrtc.DataChannelMessage) {

	})
	dcPeer1.OnOpen(func() {

	})
	dcPeer1.OnClose(func() {

	})

	dcPeer2, err := peer1.CreateDataChannel(roomName, nil)
	if err != nil {
		panic(err)
	}
	defer dcPeer2.Close()

	dcPeer2.OnMessage(func(msg webrtc.DataChannelMessage) {

	})
	dcPeer2.OnOpen(func() {

	})
	dcPeer2.OnClose(func() {

	})

	// offer, err := peer2.CreateOffer(nil)
	// if err != nil {
	// 	panic(err)
	// }
	// if err := peer2.SetLocalDescription(offer); err != nil {
	// 	panic(err)
	// }
	//
	// if err := peer1.SetRemoteDescription(offer); err != nil {
	// 	panic(err)
	// }
	// answer, err := peer1.CreateAnswer(nil)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// timeout := time.After(time.Second * 3)
	// select {
	// case <-webrtc.GatheringCompletePromise(peer2):
	// 	log.Println("Gathering complete")
	// case <-webrtc.GatheringCompletePromise(peer1):
	// 	log.Println("Gathering complete")
	// // case <-ctx.Done():
	// // 	return
	// case <-timeout:
	// 	t.Error("Timeout")
	// }
	//
	// exchangeIceCandidates(peer1, peer2)
	// exchangeIceCandidates(peer2, peer1)
	//
	// if err := peer1.SetLocalDescription(answer); err != nil {
	// 	panic(err)
	// }

	// peer1.OnNegotiationNeeded(func() {
	// 	offer, err := peer1.CreateOffer(nil)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	if err := peer1.SetLocalDescription(offer); err != nil {
	// 		panic(err)
	// 	}
	//
	// 	fmt.Println("Created offer:", &offer)
	// 	//
	//
	// 	// if err := peer2.SetRemoteDescription(offer); err != nil {
	// 	// 	panic(err)
	// 	// }
	// 	//
	// 	// answer, err := peer2.CreateAnswer(nil)
	// 	// if err != nil {
	// 	// 	panic(err)
	// 	// }
	// 	//
	// 	// if err := peer2.SetLocalDescription(answer); err != nil {
	// 	// 	panic(err)
	// 	// }
	//
	// 	// offer, err := peer1.CreateOffer(nil)
	// 	// if err != nil {
	// 	// 	t.Error(err)
	// 	// 	return
	// 	// }
	// 	// createWebrtcAnswer(peer2, offer)
	// 	done()
	// })

	// <-webrtc.GatheringCompletePromise(peer1)
	// <-webrtc.GatheringCompletePromise(peer2)

	timeout := time.After(time.Second * 3)
	//
	// // for {
	select {
	// case <-webrtc.GatheringCompletePromise(peer2):
	// 	log.Println("Gathering complete")
	// case <-webrtc.GatheringCompletePromise(peer1):
	// 	log.Println("Gathering complete")
	// // case <-ctx.Done():
	// // 	return
	case <-timeout:
		t.Error("Timeout")
	}
	// // }
}
