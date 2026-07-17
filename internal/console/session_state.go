package console

import (
	"fmt"
	"log/slog"
)

// SessionState is the explicit lifecycle state of a lobby UserSession. It is the
// single source of truth, replacing the previously implicit signals (non-nil
// WebSocket = connected, non-zero UserID = authed, map presence = in-lobby/room).
//
// StateConnecting is 0 so a zero-value UserSession starts in the correct state
// without explicit initialization (important for test helpers that build the
// struct directly).
type SessionState int32

const (
	StateConnecting SessionState = iota
	StateAuthenticating
	StateInLobby
	StateInRoom
	StateDisconnecting
	StateDisconnected
)

func (s SessionState) String() string {
	switch s {
	case StateConnecting:
		return "connecting"
	case StateAuthenticating:
		return "authenticating"
	case StateInLobby:
		return "in_lobby"
	case StateInRoom:
		return "in_room"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	default:
		return fmt.Sprintf("SessionState(%d)", int(s))
	}
}

// allowedTransitions documents valid state changes. It is intentionally NOT
// enforced by Transition: the CAS provides the exactly-once guarantee, and this
// table is used for logging/audit and tests. Disconnected is reachable from any
// state because teardown can be triggered from anywhere.
var allowedTransitions = map[SessionState][]SessionState{
	StateConnecting:     {StateAuthenticating, StateDisconnecting, StateDisconnected},
	StateAuthenticating: {StateInLobby, StateDisconnected},
	StateInLobby:        {StateInRoom, StateDisconnecting, StateDisconnected},
	StateInRoom:         {StateInLobby, StateDisconnecting, StateDisconnected},
	StateDisconnecting:  {StateDisconnected},
	// StateDisconnected: terminal, no outgoing transitions.
}

func contains(states []SessionState, target SessionState) bool {
	for _, s := range states {
		if s == target {
			return true
		}
	}
	return false
}

// Transition attempts to move the session to `to`. It returns true if the state
// was changed (the CAS won), false if the session is already terminal or another
// goroutine changed it first. Transitions are never rejected: an unexpected one
// is logged as a warning for audit but still applied.
func (us *UserSession) Transition(to SessionState) bool {
	for {
		from := us.state.Load()
		if from == int32(StateDisconnected) {
			return false
		}
		if SessionState(from) == to {
			// Self-transition is a no-op (e.g. a second Disconnecting attempt
			// must not re-arm the teardown goroutine).
			return false
		}
		if !us.state.CompareAndSwap(from, int32(to)) {
			continue // concurrent writer; retry with fresh state
		}
		if !contains(allowedTransitions[SessionState(from)], to) {
			slog.Warn("unexpected session state transition",
				"user", us.UserID, "from", SessionState(from).String(), "to", to.String())
		} else {
			slog.Debug("session state transition",
				"user", us.UserID, "from", SessionState(from).String(), "to", to.String())
		}
		return true
	}
}

// State returns the current session state.
func (us *UserSession) State() SessionState {
	return SessionState(us.state.Load())
}
