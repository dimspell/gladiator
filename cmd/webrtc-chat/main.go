package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/dimspell/gladiator/cmd/webrtc-chat/internal"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	userID := uuid.New().String()[:6]
	p, err := internal.Dial(&internal.DialParams{
		SignalingURL: "ws://localhost:5050",
		RoomName:     "test",
		ID:           userID,
	})
	if err != nil {
		panic(err)
	}

	go func() {
		rd := bufio.NewReader(os.Stdin)
		for {
			line, _, err := rd.ReadLine()
			if err != nil {
				log.Printf("Error reading message: %v", err)
				return
			}
			p.Broadcast(line)
		}
	}()

	p.Run(func(peer *internal.Peer, packet webrtc.DataChannelMessage) {
		fmt.Println("Received:", string(packet.Data))
	})
}
