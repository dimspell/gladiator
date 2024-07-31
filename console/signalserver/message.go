package signalserver

import (
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

type Member struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Channel string `json:"channel"`
}

type Offer struct {
	Name  string                    `json:"name"`
	Offer webrtc.SessionDescription `json:"offer"`
}
