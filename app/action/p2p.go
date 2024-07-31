package action

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/dimspell/gladiator/backend/proxy"
	"github.com/google/uuid"
	"github.com/urfave/cli/v3"
)

func P2PCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "p2p",
		Description: "Create a P2P connection with a game server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "turn-addr",
				Value: "192.168.121.169:38",
				Usage: "IP address of the game server hosting a game",
			},
			&cli.StringFlag{
				Name:  "turn-realm",
				Value: "dispel-multi",
				Usage: "Realm to use for TURN server",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		id := uuid.New().String()[:6]

		p2p := proxy.NewPeerToPeer("ws://localhost:5050")

		if _, err := p2p.Create("", id); err != nil {
			return err
		}
		if err := p2p.HostGame("test", proxy.User(id)); err != nil {
			return err
		}

		select {}

		// go p2p.Run(
		// 	func(peer *client.Peer, packet webrtc.DataChannelMessage) {
		// 		log.Printf("Received UDP message from %s: %s", peer.ID, string(packet.Data))
		// 	},
		// 	func(peer *client.Peer, packet webrtc.DataChannelMessage) {
		// 		log.Printf("Received TCP message from %s: %s", peer.ID, string(packet.Data))
		// 	},
		// )

		rd := bufio.NewReader(os.Stdin)
		for {
			line, _, err := rd.ReadLine()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return err
			}

			p2p.TodoBroadcast(line)
			// p2p.BroadcastTCP(line)
		}

		return nil
	}

	return cmd
}
