package backend

import (
	"context"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/dimspell/gladiator/internal/lobby"
	"go.uber.org/goleak"
)

func TestBackend_RegisterNewObserver(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lb := lobby.NewLobby(ctx)
	ts := httptest.NewServer(lb.Handler())
	defer ts.Close()

	b := &Backend{
		SignalServerURL: "ws://" + ts.URL[len("http://"):], // Skip schema prefix.
	}
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	if err := b.RegisterNewObserver(session); err != nil {
		t.Error(err)
		return
	}
	if len(conn.Written) != 0 {
		t.Error("expected no data written to the backend client")
	}
	session.observerDone()
	// _ = session.wsConn.CloseNow()
}

func TestBackend_UpdateCharacterInfo(t *testing.T) {

}
