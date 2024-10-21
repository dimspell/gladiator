package console

import (
	"context"
	"fmt"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/wire"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConsole_Handlers(t *testing.T) {
	t.Run("GET health", func(t *testing.T) {
		db, err := database.NewMemory()
		if err != nil {
			t.Error(err)
			return
		}
		c := NewConsole(db, "")
		ts := httptest.NewServer(c.HttpRouter())
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		http.DefaultClient.Timeout = time.Second

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/_health", nil)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK; got %d", resp.StatusCode)
			return
		}
	})

	t.Run("Connect to websocket", func(t *testing.T) {
		c := &Console{}
		ts := httptest.NewServer(c.HttpRouter())
		defer ts.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		uri := fmt.Sprintf("ws://%s/lobby", ts.URL[7:])
		t.Logf("connecting to %s", uri)

		_, err := wire.Connect(ctx, uri, wire.User{
			UserID:   1,
			Username: "tester",
			Version:  "dev",
		})
		if err != nil {
			t.Error(err)
		}
	})
}
