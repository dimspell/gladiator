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
