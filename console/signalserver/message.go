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

const (
	RoleHost  string = "host"
	RoleGuest string = "guest"
)

type Member struct {
	// UserID is the identifier used by the console to identify the user.
	UserID string `json:"userID"`

	// Role defines information who is the given user (RoleHost or RoleGuest)
	Role string `json:"role"`
}

type Offer struct {
	Member Member                    `json:"member"`
	Offer  webrtc.SessionDescription `json:"offer"`
}
