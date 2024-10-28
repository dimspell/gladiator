package wire

type EventType uint

const (
	_ EventType = iota
	Hello
	Welcome
	LobbyUsers
	JoinLobby
	JoinedLobby
	LeaveLobby
	Chat
	CreateRoom
	SetRoomReady
	JoinRoom
	LeaveRoom
	RTCOffer
	RTCAnswer
	RTCICECandidate
)

func (e EventType) String() string {
	switch e {
	case Hello:
		return "Hello"
	case Welcome:
		return "Welcome"
	case LobbyUsers:
		return "LobbyUsers"
	case JoinLobby:
		return "JoinLobby"
	case JoinedLobby:
		return "JoinedLobby"
	case LeaveLobby:
		return "LeaveLobby"
	case Chat:
		return "Chat"
	case CreateRoom:
		return "CreateRoom"
	case SetRoomReady:
		return "SetRoomReady"
	case JoinRoom:
		return "JoinRoom"
	case LeaveRoom:
		return "LeaveRoom"
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
