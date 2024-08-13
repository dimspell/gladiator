package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/dimspell/gladiator/cmd/webrtc-chat/internal"
	"github.com/google/uuid"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-colorable"
	"github.com/pion/webrtc/v4"
)

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(
			colorable.NewColorable(os.Stderr),
			&tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
				AddSource:  true,
			},
		),
	))

	userID := uuid.New().String()[:6]
	p, err := internal.Dial(&internal.DialParams{
		SignalingURL: "ws://localhost:5050",
		RoomName:     "test",
		ID:           userID,
		Name:         userID,
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
