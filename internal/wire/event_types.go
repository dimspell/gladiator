package wire

type EventType uint

const (
	_ EventType = iota
	Hello
	Welcome
	LobbyUsers
	Join
	Joined
	Leave
	RTCOffer
	RTCAnswer
	RTCICECandidate
	Chat
)

func (e EventType) String() string {
	switch e {
	case Hello:
		return "Hello"
	case Welcome:
		return "Welcome"
	case LobbyUsers:
		return "LobbyUsers"
	case Join:
		return "Join"
	case Joined:
		return "Joined"
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
