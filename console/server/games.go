package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/go-chi/render"
)

// ListActiveGames lists all game rooms that are currently active.
func (c *ConsoleServer) ListActiveGames(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	gameRooms, err := c.DB.ListGameRooms(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, r, gameRooms)
}

func (c *ConsoleServer) CreateGame(w http.ResponseWriter, r *http.Request) {
	type Input struct {
		RoomName      string
		RoomPassword  string
		HostIPAddress string
		MapID         int64
	}

	var input Input
	json.NewDecoder(r.Body).Decode(&input)

	newGameRoom, _ := c.DB.CreateGameRoom(context.TODO(), database.CreateGameRoomParams{
		Name:          input.RoomName,
		Password:      sql.NullString{String: input.RoomPassword, Valid: len(input.RoomPassword) > 0},
		HostIpAddress: input.HostIPAddress,
		MapID:         input.MapID,
	})

	render.JSON(w, r, newGameRoom)
}
