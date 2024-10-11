package icesignal

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
	Chat
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
	case Chat:
		return "Chat"
	default:
		return "Unknown"
	}
}
