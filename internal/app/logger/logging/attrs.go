package logging

import (
	"log/slog"
)

func Error(err error) slog.Attr {
	if err == nil {
		slog.Error("Going to log nil error")
		return slog.String("error", "<nil>")
	}
	return slog.String("error", err.Error())
}

func RoomID(roomID string) slog.Attr {
	return slog.String("roomId", roomID)
}

func PeerID(peerID string) slog.Attr {
	return slog.String("peerId", peerID)
}
