package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/redirect"
)

type Host struct {
	NotUsedIP string
	HostType  string
	UDPPort   int
	TCPPort   int

	peerID   string
	fakeHost *redirect.FakeHost
}

var variant1 = map[string]*Host{
	"player2": {
		NotUsedIP: "127.0.2.1",
		HostType:  "LISTEN",
		UDPPort:   5023,
		TCPPort:   5024,
	},
	"player3": {
		NotUsedIP: "127.0.3.1",
		HostType:  "LISTEN",
		UDPPort:   5033,
		// TCPPort:   5034,
	},
	"player4": {
		NotUsedIP: "127.0.4.1",
		HostType:  "LISTEN",
		UDPPort:   5043,
		// TCPPort:   5044,
	},
}

func main() {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx := context.Background()

	// r := relay.PacketRouter{}

	hm := redirect.NewManager(net.IPv4(127, 0, 0, 1))

	for peerID, params := range variant1 {
		ip, _ := hm.AssignIP(peerID)

		h, err := hm.CreateFakeHost(ctx,
			"TEST",
			peerID,
			ip,
			&redirect.ProxySpec{
				LocalIP: "127.0.0.1",
				Port:    params.TCPPort,
				Create: func(ipv4, port string) (redirect.Redirect, error) {
					if params.HostType == "LISTEN" {
						return redirect.ListenTCP(ipv4, port)
					}
					if params.HostType == "DIAL" {
						return redirect.DialTCP(ipv4, port)
					}
					return nil, fmt.Errorf("unknown host type %s", params.HostType)
				},
				OnReceive: func(p []byte) error {
					slog.Info("[TCP] Received", "data", string(p))
					return nil
				},
			},
			&redirect.ProxySpec{
				LocalIP: "127.0.0.1",
				Port:    params.UDPPort,
				Create: func(ipv4, port string) (redirect.Redirect, error) {
					if params.HostType == "LISTEN" {
						return redirect.ListenUDP(ipv4, port)
					}
					if params.HostType == "DIAL" {
						return redirect.DialUDP(ipv4, port)
					}
					return nil, fmt.Errorf("unknown host type %s", params.HostType)
				},
				OnReceive: func(p []byte) error {
					slog.Info("[UDP] Received", "data", string(p))
					return nil
				},
			},
			func(host *redirect.FakeHost) {
				fmt.Println("Disconnecting", peerID, host)
				hm.StopHost(host)
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		params.peerID = peerID
		params.fakeHost = h
	}

	// <-time.After(1 * time.Second)
	// h := hm.Hosts["127.0.0.2"]
	// hm.StopHost(h)

	select {}

	// 	go func() {
	// 		for {
	// 			t := h.ProxyTCP.(*redirect.ListenerTCP)
	// 			fmt.Println(t.Alive(time.Now(), 5*time.Second))
	// 			time.Sleep(2 * time.Second)
	// 		}
	// 	}()
}
