package action

import (
	"bufio"
	"context"
	"log"
	"os"

	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
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
			&cli.StringFlag{
				Name:  "mode",
				Value: "host",
			},
		},
	}

	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		id := uuid.New().String()[:6]

		peerToPeer := proxy.NewPeerToPeer("ws://localhost:5050")

		// ch := make(chan string)
		//
		// go func() {
		// 	for {
		// 		select {
		// 		case <-ch:
		// 			// nothing
		// 		}
		// 	}
		// }()

		if c.String("mode") == "host" {
			if _, err := peerToPeer.Create(proxy.CreateParams{
				HostUserIP: "",
				GameID:     "test",
				HostUserID: "host1",
			}); err != nil {
				return err
			}
			if err := peerToPeer.Host(proxy.HostParams{
				GameID:     "test",
				HostUserID: "host1",
			}); err != nil {
				return err
			}
		} else {
			if ip, err := peerToPeer.Join(proxy.JoinParams{
				HostUserID:    "host1",
				CurrentUserID: id,
				GameID:        "test",
				CurrentUserIP: "",
			}); err != nil {
				return err
			} else {
				log.Printf("Joined game at %s", ip)
			}
		}

		rd := bufio.NewReader(os.Stdin)
		for {
			_, _, err := rd.ReadLine()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return err
			}

			peerToPeer.Peers.Range(func(s string, peer *p2p.Peer) {
				log.Println(s, peer)
			})
			// ch <- string(line)
		}

		return nil
	}

	return cmd
}
