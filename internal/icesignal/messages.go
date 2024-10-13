package icesignal

import (
	"github.com/pion/webrtc/v4"
)

const (
	SupportedRealm = "signalserver"
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

type Player struct {
	ID                 string
	Username           string
	CharacterID        string
	CharacterClassType int
}

type ChatMessage struct {
	User string
	Text string
}
