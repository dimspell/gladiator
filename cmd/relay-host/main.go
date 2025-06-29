package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
	"github.com/dimspell/gladiator/internal/model"
)

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	px := &relay.ProxyRelay{
		RelayServerAddr: "localhost:9999",
	}
	session := bsession.NewSession(nil)
	session.ID = "sid-1"
	session.UserID = 1
	session.Username = "knight"
	session.CharacterID = 1
	session.ClassType = model.ClassTypeKnight
	session.Proxy = px.Create(session)

	ctx := context.TODO()

	if err := session.ConnectOverWebsocket(ctx, &multiv1.User{
		UserId:   session.UserID,
		Username: session.Username,
	}, fmt.Sprintf("ws://%s/lobby", "localhost:2137")); err != nil {
		slog.Error("failed to connect over websocket", logging.Error(err))
		return
	}
	if err := session.JoinLobby(ctx); err != nil {
		slog.Error("JoinLobby", logging.Error(err))
		return
	}

	go func(ctx context.Context) {
		for {
			p, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, logging.Error(err))
				return
			}
			if err := session.Proxy.Handle(ctx, p); err != nil {
				slog.Error("Error handling message", "session", session.ID, logging.Error(err))
				return
			}
		}
	}(ctx)

	roomID := "testroom"

	var err error

	_, err = session.Proxy.CreateRoom(proxy.CreateParams{
		GameID: roomID,
	})
	if err != nil {
		slog.Error("CreateRoom", logging.Error(err))
		return
	}

	consoleUri := fmt.Sprintf("%s://%s/grpc", "http", "localhost:2137")
	gameClient := multiv1connect.NewGameServiceClient(&http.Client{Timeout: 10 * time.Second}, consoleUri)
	if _, err := gameClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      roomID,
		Password:      "",
		MapId:         multiv1.GameMap(1),
		HostUserId:    session.UserID,
		HostIpAddress: "127.0.0.1",
	})); err != nil {
		slog.Error("CreateGame", logging.Error(err))
		return
	}

	// startFakeBackendServer(ctx)

	err = session.Proxy.HostRoom(ctx, proxy.HostParams{GameID: roomID})
	if err != nil {
		slog.Error("HostRoom", logging.Error(err))
		return
	}

	select {}
}

func startFakeBackendServer(ctx context.Context) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:6114")
	if err != nil {
		slog.Error("fake backend server failed", logging.Error(err))
		os.Exit(1)
		return
	}

	ln, err := net.Listen("tcp", tcpAddr.String())
	if err != nil {
		return
	}

	srcAddr, err := net.ResolveUDPAddr("udp", "localhost:6113")
	if err != nil {
		return
	}
	srcConn, err := net.ListenUDP("udp", srcAddr)
	if err != nil {
		return
	}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				defer c.Close()

				for {
					buf := make([]byte, 1024)

					n, err := c.Read(buf)
					if err != nil {
						return
					}

					msg := buf[:n]
					slog.Debug("[GameServer] Received TCP", "message", msg)

					// TODO: Write back
				}
			}()
		}
	}()

	go func() {
		buf := make([]byte, 1024)

		for {
			n, _, err := srcConn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			msg := buf[:n]
			slog.Debug("[GameServer] Received UDP", "message", msg)
		}
	}()
}
