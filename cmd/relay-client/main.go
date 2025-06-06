package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io"
	"log"
	"time"

	"github.com/quic-go/quic-go"
)

var hmacKey = []byte("shared-secret-key")

func sign(data []byte) []byte {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(data)
	return append(mac.Sum(nil), data...)
}

func verify(packet []byte) ([]byte, bool) {
	if len(packet) < 32 {
		return nil, false
	}
	sig := packet[:32]
	data := packet[32:]
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(data)
	expected := mac.Sum(nil)
	return data, hmac.Equal(sig, expected)
}

type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", "data", "broadcast"
	RoomID  string `json:"room"`
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

func main() {
	var clientID string
	flag.StringVar(&clientID, "name", "clientA", "name of the client")
	flag.Parse()

	serverAddr := "localhost:9999"

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
	}
	conn, err := quic.DialAddr(context.Background(), serverAddr, tlsConf, nil)
	if err != nil {
		log.Fatalf("Dial failed: %v", err)
	}
	defer conn.CloseWithError(0, "done")

	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		log.Fatalf("Stream open failed: %v", err)
	}

	// Send signed join packet
	join := RelayPacket{
		Type:   "join",
		RoomID: "room123",
		FromID: clientID,
	}
	sendPacket(stream, join)

	// Start receiver
	go receiveLoop(stream)

	// Send a broadcast
	time.Sleep(1 * time.Second)
	broadcast := RelayPacket{
		Type:    "broadcast",
		RoomID:  "room123",
		FromID:  clientID,
		Payload: []byte("Hello everyone from clientA in room123!"),
	}
	sendPacket(stream, broadcast)

	// Optional: leave after 10s
	// time.Sleep(10 * time.Second)
	// sendPacket(stream, RelayPacket{Type: "leave", FromID: clientID})

	// fmt.Println("Client exiting.")
	select {}
}

func sendPacket(stream quic.Stream, pkt RelayPacket) {
	data, err := json.Marshal(pkt)
	if err != nil {
		log.Printf("Marshal error: %v", err)
		return
	}
	packet := sign(data)
	_, err = stream.Write(packet)
	if err != nil {
		log.Printf("Send error: %v", err)
	}
}

func receiveLoop(stream quic.Stream) {
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Receive error: %v", err)
			return
		}
		data, ok := verify(buf[:n])
		if !ok {
			log.Println("Invalid signature in received packet")
			continue
		}
		var pkt RelayPacket
		if err := json.Unmarshal(data, &pkt); err != nil {
			log.Printf("JSON error: %v", err)
			continue
		}
		log.Printf("Received from %s: %s", pkt.FromID, string(pkt.Payload))
	}
}
