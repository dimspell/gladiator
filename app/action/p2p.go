package action

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/dimspell/gladiator/proxy/client"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
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

		p2p, err := client.Dial(&client.DialParams{
			SignalingURL: "ws://localhost:5050",
			RoomName:     "test",
			ID:           id,
			Name:         id,
		})
		if err != nil {
			return err
		}
		defer p2p.Close()

		go p2p.Run(
			func(peer *client.Peer, packet webrtc.DataChannelMessage) {
				log.Printf("Received UDP message from %s: %s", peer.ID, string(packet.Data))
			},
			func(peer *client.Peer, packet webrtc.DataChannelMessage) {
				log.Printf("Received TCP message from %s: %s", peer.ID, string(packet.Data))
			},
		)

		rd := bufio.NewReader(os.Stdin)
		for {
			line, _, err := rd.ReadLine()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return err
			}

			p2p.BroadcastUDP(line)
			p2p.BroadcastTCP(line)
		}

		return nil
	}

	return cmd
}
