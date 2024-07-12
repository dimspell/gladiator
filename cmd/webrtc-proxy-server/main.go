package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pion/ice/v3"
	"github.com/pion/webrtc/v4"
)

var api *webrtc.API // nolint

func doSignaling(w http.ResponseWriter, r *http.Request) {
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Send the current time via a DataChannel to the remote peer every 3 seconds
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnOpen(func() {
			for range time.Tick(time.Second * 3) {

				if err = d.SendText(fmt.Sprintf("%s label: %s", time.Now().String(), d.Label())); err != nil {
					if errors.Is(io.ErrClosedPipe, err) {
						return
					}
					panic(err)
				}
			}
		})
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Println(msg.Data)
		})
	})

	var offer webrtc.SessionDescription
	if err = json.NewDecoder(r.Body).Decode(&offer); err != nil {
		panic(err)
	}

	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		panic(err)
	}

	// Block until ICE Gathering is complete, disabling trickle ICE
	// we do this because we only can exchange one signaling message
	// in a production application you should exchange ICE Candidates via OnICECandidate
	<-gatherComplete

	response, err := json.Marshal(*peerConnection.LocalDescription())
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(response); err != nil {
		panic(err)
	}
}

func main() {
	settingEngine := webrtc.SettingEngine{}

	var connectionType = "tcp"

	switch connectionType {
	case "tcp":
		// Enable support only for TCP ICE candidates.
		settingEngine.SetNetworkTypes([]webrtc.NetworkType{
			webrtc.NetworkTypeTCP4,
			webrtc.NetworkTypeTCP6,
		})

		tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.IP{0, 0, 0, 0},
			Port: 8443,
		})
		if err != nil {
			panic(err)
		}

		fmt.Printf("Listening for ICE TCP at %s\n", tcpListener.Addr())

		tcpMux := webrtc.NewICETCPMux(nil, tcpListener, 8)
		settingEngine.SetICETCPMux(tcpMux)
	case "udp":
		// Configure our SettingEngine to use our UDPMux. By default, a PeerConnection has
		// no global state. The API+SettingEngine allows the user to share state between them.
		// In this case we are sharing our listening port across many.
		// Listen on UDP Port 8443, will be used for all WebRTC traffic
		mux, err := ice.NewMultiUDPMuxFromPort(8443)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Listening for WebRTC traffic at %d\n", 8443)
		settingEngine.SetICEUDPMux(mux)
	}

	// Create a new API using our SettingEngine
	api = webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/doSignaling", doSignaling)

	fmt.Println("Open http://localhost:8080 to access this demo")
	// nolint: gosec
	panic(http.ListenAndServe(":8080", nil))
}
