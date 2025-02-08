package backend

import (
	"context"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

type LobbyEventHandler struct {
	Session *bsession.Session
}

// NewLobbyEventHandler creates a new LobbyEventHandler for the given Session.
func NewLobbyEventHandler(session *bsession.Session) *LobbyEventHandler {
	return &LobbyEventHandler{session}
}

func (h *LobbyEventHandler) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	switch eventType {
	case wire.Chat:
		_, msg, err := wire.DecodeTyped[wire.ChatMessage](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", eventType.String(), "payload", payload)
			return nil
		}
		// if err := session.Send(ReceiveMessage, NewGlobalMessage(msg.Content.User, msg.Content.Text)); err != nil {
		if err := h.Session.Send(packet.ReceiveMessage, NewLobbyMessage(msg.Content.User, msg.Content.Text)); err != nil {
			slog.Error("Error writing chat message over the backend wire", "session", h.Session.ID, "error", err)
			return nil
		}
	case wire.LobbyUsers:
		// TODO: Handle it. Note: It should be sent only once.
		_, msg, err := wire.DecodeTyped[[]wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", eventType.String(), "payload", payload)
			return nil
		}

		h.Session.State.UpdateLobbyUsers(msg.Content)
	case wire.JoinLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", eventType.String(), "payload", payload)
			return nil
		}
		if msg.Content.UserID == h.Session.UserID {
			return nil
		}

		player := msg.Content
		lobbyUsers := append(h.Session.State.GetLobbyUsers(), player)
		h.Session.State.UpdateLobbyUsers(lobbyUsers)
		idx := uint32(len(lobbyUsers))

		if err := h.Session.Send(packet.ReceiveMessage, AppendCharacterToLobby(player.Username, model.ClassType(player.ClassType), idx)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.Session.ID, "error", err)
			return nil
		}
	case wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", eventType.String(), "payload", payload)
			return nil
		}

		h.Session.State.DeleteLobbyUser(msg.Content.UserID)

		if err := h.Session.Send(packet.ReceiveMessage, RemoveCharacterFromLobby(msg.Content.Username)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.Session.ID, "error", err)
			return nil
		}
	default:
		// Skip and do not handle it.
	}

	return nil
}
