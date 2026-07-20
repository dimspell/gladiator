package console

import (
	"github.com/dimspell/gladiator/internal/relayserver"
)

// RelayListener is the transport seam for accepting relay connections. It
// decouples RelayServer from *quic.Listener so an in-memory implementation can
// be injected in tests (and the relay could later run over a different
// transport) without touching the relay logic.
type RelayListener = relayserver.RelayListener

// WithListener injects a RelayListener, overriding the default real QUIC
// listener created by NewRelayServer. Used by tests to run the relay server
// fully in-process.
func WithListener(l RelayListener) RelayServerOption {
	return relayserver.WithListener(l)
}
