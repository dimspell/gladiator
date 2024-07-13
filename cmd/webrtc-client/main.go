package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/pion/randutil"
	"github.com/pion/webrtc/v4"
	"golang.org/x/net/websocket"
)

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(
			colorable.NewColorable(os.Stderr),
			&tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
			},
		),
	))

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// {
			// 	URLs: []string{"stun:stun.l.google.com:19302"},
			// },
			// {
			// 	URLs:       []string{"turn:127.0.0.1:3478"},
			// 	Username:   "username2",
			// 	Credential: "password2",
			// },
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			fmt.Println("Peer Connection has gone to closed exiting")
			os.Exit(0)
		}
	})

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Println(dataChannel, err)
	}

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

		for range time.NewTicker(5 * time.Second).C {
			message, sendTextErr := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			if sendTextErr != nil {
				panic(sendTextErr)
			}

			// Send the message as text
			fmt.Printf("Sending '%s'\n", message)
			if sendTextErr = dataChannel.SendText(message); sendTextErr != nil {
				panic(sendTextErr)
			}
		}
	})

	// dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
	// 	slog.Info("Message from DataChannel", dataChannel.Label(), string(msg.Data))
	// })

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open\n", d.Label(), d.ID())

			// for range time.NewTicker(5 * time.Second).C {
			// 	message, sendErr := randutil.GenerateCryptoRandomString(15, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			// 	if sendErr != nil {
			// 		panic(sendErr)
			// 	}
			//
			// 	// Send the message as text
			// 	fmt.Printf("Sending '%s'\n", message)
			// 	if sendErr = d.SendText(message); sendErr != nil {
			// 		panic(sendErr)
			// 	}
			// }
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})

	u := url.URL{Scheme: "ws", Host: net.JoinHostPort("127.0.0.1", "2137"), Path: "/ws"}
	ws, err := websocket.Dial(u.String(), "", "http://127.0.0.1")
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer ws.Close()

	pendingCandidates := []*webrtc.ICECandidate{}
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		fmt.Printf("Local ICE Candidate: %v\n", candidate)
		pendingCandidates = append(pendingCandidates, candidate)
	})

	// Create an offer to send to the other process
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Send our offer to the HTTP server listening in the other process
	payload, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sending offer: %s\n", payload)
	if _, err := ws.Write(payload); err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	go func() {
		buf := make([]byte, 1500)
		for {
			// Read each inbound WebSocket Message
			n, err := ws.Read(buf)
			if err != nil {
				panic(err)
			}

			// Unmarshal each inbound WebSocket message
			var (
				candidate webrtc.ICECandidateInit
				answer    webrtc.SessionDescription
			)

			switch {
			// Attempt to unmarshal as a SessionDescription. If the SDP field is empty
			// assume it is not one.
			case json.Unmarshal(buf[:n], &answer) == nil && answer.SDP != "":
				slog.Info("Received Answer", "answer", answer)

				if err = peerConnection.SetRemoteDescription(answer); err != nil {
					panic(err)
				}

				for _, candidate := range pendingCandidates {
					payload, err := json.Marshal(candidate.ToJSON())
					if err != nil {
						panic(err)
					}

					if _, err := ws.Write(payload); err != nil {
						panic(err)
					}
				}

			// Attempt to unmarshal as a ICECandidateInit. If the candidate field is empty
			// assume it is not one.
			case json.Unmarshal(buf[:n], &candidate) == nil && candidate.Candidate != "":
				slog.Info("Received ICE Candidate", "candidate", candidate)

				if err = peerConnection.AddICECandidate(candidate); err != nil {
					panic(err)
				}
			default:
				log.Println(string(buf[:n]))
				panic("Unknown message")
			}
		}
	}()

	select {}
}
