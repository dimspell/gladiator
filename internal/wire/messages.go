package wire

import (
	"fmt"

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

type User struct {
	UserID   int64  `json:"userID"`
	Username string `json:"username"`
	Version  string `json:"version"`
}

func (u *User) ID() string {
	return fmt.Sprint(u.UserID)
}

type Character struct {
	CharacterID int64 `json:"characterID"`
	ClassType   byte  `json:"classType"`
}

type Player struct {
	UserID      int64  `json:"userID"`
	Username    string `json:"username"`
	CharacterID int64  `json:"characterID"`
	ClassType   byte   `json:"classType"`

	IPAddress string `json:"ipAddress"`
}

func (p *Player) ID() string {
	return fmt.Sprint(p.UserID)
}

type ChatMessage struct {
	User string
	Text string
}
