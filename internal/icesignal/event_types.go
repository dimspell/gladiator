package icesignal

type EventType uint

const (
	_ EventType = iota
	HandshakeRequest
	LobbyUsers
	Join
	Leave
	RTCOffer
	RTCAnswer
	RTCICECandidate
	Chat
)

func (e EventType) String() string {
	switch e {
	case HandshakeRequest:
		return "HandshakeRequest"
	case LobbyUsers:
		return "LobbyUsers"
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
	case Chat:
		return "Chat"
	default:
		return "Unknown"
	}
}
