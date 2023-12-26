package pubsub

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

type JoinGameRequest struct {
	GameId int64
}

func PublishJoinGame(nc *nats.Conn, req JoinGameRequest) error {
	if req.GameId == 0 {
		return nil
	}

	msg := &nats.Msg{
		Subject: fmt.Sprintf("game.%d", req.GameId),
		Reply:   "",
		Header:  nil,
		Data:    nil,
		Sub:     nil,
	}

	if err := nc.PublishMsg(msg); err != nil {
		return err
	}
	return nil
}
