package signalserver

import (
	"encoding/json"

	"github.com/fxamacker/cbor/v2"
	"github.com/pion/webrtc/v4"
)

// Message represents a chat message
type Message struct {
	To      string    `json:"to,omitempty"`
	From    string    `json:"from,omitempty"`
	Type    EventType `json:"type"`
	Content any       `json:"content"`
}

type MessageContent[T any] struct {
	From    string    `json:"from"`
	Type    EventType `json:"type"`
	Content T         `json:"content"`
	To      string    `json:"to,omitempty"`
}

type EventType uint

const (
	_ EventType = iota
	HandshakeRequest
	HandshakeResponse
	Join
	Leave
	RTCOffer
	RTCAnswer
	RTCICECandidate
)

func (m Message) ToCBOR() []byte {
	out, err := cbor.Marshal(m)
	if err != nil {
		panic(err)
	}
	return out
}

func (m Message) ToJSON() []byte {
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return out
}

type Member struct {
	// UserID is the identifier used by the console to identify the user.
	UserID string `json:"userID"`

	IsHost bool `json:"isHost"`

	// Joined defines whether the given user has already joined to the room.
	Joined bool `json:"joined"`
}

type Offer struct {
	Member Member                    `json:"member"`
	Offer  webrtc.SessionDescription `json:"offer"`
}
