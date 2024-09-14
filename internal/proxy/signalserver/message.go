package signalserver

import (
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

func (e EventType) String() string {
	switch e {
	case HandshakeRequest:
		return "HandshakeRequest"
	case HandshakeResponse:
		return "HandshakeResponse"
	case Join:
		return "Join"
	case Leave:
		return "Leave"
	case RTCOffer:
		return "RTCOffer"
	case RTCAnswer:
		return "RTCAnswer"
	case RTCICECandidate:
		return "RTCICECandidate"
	default:
		return "Unknown"
	}
}

func (m Message) Encode() []byte {
	out, err := DefaultCodec.Marshal(m)
	if err != nil {
		panic(err)
	}
	return out
}

type Member struct {
	// UserID is the identifier used by the console to identify the user.
	UserID string `json:"userID"`

	IsHost bool `json:"isHost"`
}

type Offer struct {
	Member Member                    `json:"member"`
	Offer  webrtc.SessionDescription `json:"offer"`
}
