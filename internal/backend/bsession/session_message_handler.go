package bsession

import (
	"context"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/packet/command"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

type SessionMessageHandler struct {
	Session *Session
}

func (h *SessionMessageHandler) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)
	// msg := wire.ParseTyped[wire.ChatMessage](payload)

	// handleWireEvent := func(et wire.EventType, p []byte) {
	switch et {
	case wire.Chat:
		_, msg, err := wire.DecodeTyped[wire.ChatMessage](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}
		// if err := session.Send(ReceiveMessage, NewGlobalMessage(msg.Content.User, msg.Content.Text)); err != nil {
		if err := h.Session.Send(command.ReceiveMessage, command.NewSystemMessage(msg.Content.User, msg.Content.Text, "???")); err != nil {
			slog.Error("Error writing chat message over the backend wire", "session", h.Session.ID, "error", err)
			return nil
		}
	case wire.LobbyUsers:
		// TODO: Handle it. Note: It should be sent only once.
		_, msg, err := wire.DecodeTyped[[]wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}

		h.Session.State.UpdateLobbyUsers(msg.Content)
	case wire.JoinLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}
		if msg.Content.UserID == h.Session.UserID {
			return nil
		}

		player := msg.Content
		lobbyUsers := append(h.Session.State.GetLobbyUsers(), player)
		h.Session.State.UpdateLobbyUsers(lobbyUsers)
		idx := uint32(len(lobbyUsers))

		if err := h.Session.Send(command.ReceiveMessage, command.AppendCharacterToLobby(player.Username, model.ClassType(player.ClassType), idx)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.Session.ID, "error", err)
			return nil
		}
	case wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			slog.Warn("Could not decode the message", "session", h.Session.ID, "error", err, "event", et.String(), "payload", payload)
			return nil
		}

		h.Session.State.DeleteLobbyUser(msg.Content.UserID)

		if err := h.Session.Send(command.ReceiveMessage, command.RemoveCharacterFromLobby(msg.Content.Username)); err != nil {
			slog.Warn("Error appending lobby user", "session", h.Session.ID, "error", err)
			return nil
		}
	default:
		// Skip and do not handle it.
	}

	return nil
}
