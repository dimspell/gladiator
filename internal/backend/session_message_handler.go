package backend

import (
	"context"
	"log/slog"

	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

type WireMessageHandler struct {
	session *Session
}

func (h *WireMessageHandler) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)
	// msg := wire.ParseTyped[wire.ChatMessage](payload)

	// handleWireEvent := func(et wire.EventType, p []byte) {
	switch et {
	case wire.Chat:
		_, msg, err := wire.DecodeTyped[wire.ChatMessage](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}
		// if err := session.Send(ReceiveMessage, NewGlobalMessage(msg.Content.User, msg.Content.Text)); err != nil {
		if err := h.session.Send(ReceiveMessage, NewSystemMessage(msg.Content.User, msg.Content.Text, "???")); err != nil {
			slog.Error("Error writing chat message over the backend wire", "session", h.session.ID, "error", err)
			return nil
		}
	case wire.LobbyUsers:
		// TODO: Handle it. Note: It should be sent only once.
		_, msg, err := wire.DecodeTyped[[]wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}

		h.session.State.UpdateLobbyUsers(msg.Content)
	case wire.JoinLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}
		if msg.Content.UserID == h.session.UserID {
			return nil
		}

		player := msg.Content
		lobbyUsers := append(h.session.State.GetLobbyUsers(), player)
		h.session.State.UpdateLobbyUsers(lobbyUsers)
		idx := uint32(len(lobbyUsers))

		if err := h.session.Send(ReceiveMessage, AppendCharacterToLobby(player.Username, model.ClassType(player.ClassType), idx)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.session.ID, "error", err)
			return nil
		}
	case wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}

		h.session.State.DeleteLobbyUser(msg.Content.UserID)

		if err := h.session.Send(ReceiveMessage, RemoveCharacterFromLobby(msg.Content.Username)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.session.ID, "error", err)
			return nil
		}
	default:
		// Skip and do not handle it.
	}

	return nil
}
