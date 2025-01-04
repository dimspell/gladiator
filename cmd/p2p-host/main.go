package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
)

const (
	consoleUri = "127.0.0.1:2137"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx := context.Background()

	gm := multiv1connect.NewGameServiceClient(httpClient, fmt.Sprintf("http://%s/grpc", consoleUri))
	px := proxy.NewPeerToPeer()

	session := &bsession.Session{
		ID:          "1",
		UserID:      1,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	user1 := &multiv1.User{
		UserId:   1,
		Username: "host",
	}

	if err := session.ConnectOverWebsocket(ctx, user1, fmt.Sprintf("ws://%s/lobby", consoleUri)); err != nil {
		slog.Error("failed to connect over websocket", "error", err)
		return
	}
	slog.Info("connected over websocket")

	if err := session.JoinLobby(ctx); err != nil {
		slog.Error("failed to join lobby over websocket", "error", err)
		return
	}
	slog.Info("joined lobby over websocket")

	go func() {
		handler := px.ExtendWire(session)

		for {
			payload, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				slog.Error("failed to consume websocket", "error", err)
				return
			}
			if err := handler.Handle(ctx, payload); err != nil {
				slog.Error("failed to handle websocket", "error", err)
				return
			}
		}
	}()

	game, err := gm.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      "testing",
		Password:      "",
		MapId:         0,
		HostUserId:    1,
		HostIpAddress: "127.0.1.2", // Not used for P2P traffic
	}))
	if err != nil {
		slog.Error("failed to create game over console", "error", err)
		return
	}
	slog.Info("created game over console")

	if _, err := px.CreateRoom(proxy.CreateParams{GameID: game.Msg.Game.GameId}, session); err != nil {
		slog.Error("failed to create room over proxy", "error", err)
		return
	}
	slog.Info("created room over proxy")

	if err := px.HostRoom(proxy.HostParams{GameID: game.Msg.Game.GameId}, session); err != nil {
		slog.Error("failed to host room over proxy", "error", err)
		return
	}
	slog.Info("created a host room over proxy")

	helperStartGameServer()
}

func helperStartGameServer() {
	// Listen for incoming connections.
	tcpListener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "6114"))
	if err != nil {
		log.Fatal(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", "6113"))
	if err != nil {
		log.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	// Listen UDP
	go func() {
		for {
			buf := make([]byte, 1024)
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				break
			}

			if buf[0] == '#' {
				resp := append([]byte{27, 0}, buf[1:n]...)
				_, err := udpConn.WriteToUDP(resp, udpAddr)
				if err != nil {
					slog.Debug("Failed to write to UDP", "error", err)
					return
				}
				slog.Debug("UDP response", "response", string(resp))
			}
		}
	}()

	processPackets := func(conn net.Conn) {
		slog.Info("Someone has connected over the TCP")

		message := make(chan []byte, 1)

		go func() {
			defer conn.Close()

			for {
				select {
				case msg, ok := <-message:
					if !ok {
						return
					}
					slog.Debug("message received", "msg", string(msg))
					conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
				}
			}
		}()

		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				close(message)
				return
			}
			message <- buf[:n]
		}
	}

	// ctx, cancel := context.WithCancel(context.TODO())
	for {
		// Listen for an incoming connection.
		conn, err := tcpListener.Accept()
		if err != nil {
			continue
		}
		go processPackets(conn)
	}
}
