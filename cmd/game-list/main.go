package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
)

const (
	consoleUri = "127.0.0.1:2137"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx := context.Background()

	gm := multiv1connect.NewGameServiceClient(httpClient, fmt.Sprintf("http://%s/grpc", consoleUri))

	list, err := gm.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Game list (%d)\n", len(list.Msg.Games))
	fmt.Println("--------------------------------")

	for _, g := range list.Msg.Games {
		game, err := gm.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
			GameRoomId: g.GetGameId(),
		}))
		if err != nil {
			continue
		}

		fmt.Println("Game")
		b, err := json.MarshalIndent(game, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}
}
