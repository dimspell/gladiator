package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dispel-re/dispel-multi/backend"
)

func main() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	payload := []byte{
		1, 0, 0, 0, // Is host?
		127, 0, 0, 1, // IP - recipient address to migrate
	}
	// payload := backend.NewLobbyMessage("admin", "user-hello")
	// payload := backend.NewSystemMessage("admin", "user-hello", "tos")

	packet := backend.CustomPacket{
		PacketID: uint8(backend.ChangeHost),
		Data:     payload,
	}

	// 44 (UpdateInventory) will trigger 108 (UpdateStats)
	// 108 will trigger 73 (UpdateSpells)

	input, err := json.Marshal(&packet)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:6110/send", bytes.NewReader(input))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(string(body), resp.StatusCode, resp.Status)
}
