package logging

import (
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
)

func SessionID(session *bsession.Session) slog.Attr {
	return slog.String("sessionId", session.ID)
}

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func RoomID(roomID string) slog.Attr {
	return slog.String("roomId", roomID)
}

func PeerID(peerID string) slog.Attr {
	return slog.String("peerId", peerID)
}
