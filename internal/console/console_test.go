package console

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/stretchr/testify/assert"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
}

func helperGetJSON(ctx context.Context, link string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}
	return body, nil
}

func TestConsole_Handlers(t *testing.T) {
	t.Run("GET /_health", func(t *testing.T) {
		db, err := database.NewMemory()
		if err != nil {
			t.Error(err)
			return
		}
		c := NewConsole(db)
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

	t.Run("GET /.well-known/console.json", func(t *testing.T) {
		// Arrange
		options := []Option{
			WithVersion("2.13.7"),
			WithConsoleAddr("127.0.0.1:2137", "https://console.example.com"),
			WithRelayAddr("0.0.0.0:9999", "relay.example.com:9123"),
		}

		c := NewConsole(nil, options...)
		ts := httptest.NewServer(c.HttpRouter())
		defer ts.Close()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		// Act
		http.DefaultClient.Timeout = time.Second
		body, err := helperGetJSON(ctx, ts.URL+"/.well-known/console.json")
		if err != nil {
			t.Error(err)
			return
		}
		var wellKnown model.WellKnown
		if err := json.Unmarshal(body, &wellKnown); err != nil {
			t.Error(err)
			return
		}

		// Assert
		assert.Equal(t, c.Config.ConsoleBindAddr, "127.0.0.1:2137")
		assert.Equal(t, c.Config.RelayBindAddr, "0.0.0.0:9999")

		assert.Equal(t, wellKnown.Version, "2.13.7")
		assert.Equal(t, wellKnown.Addr, "https://console.example.com")
		assert.Equal(t, wellKnown.RunMode, model.RunModeRelay)
		assert.Equal(t, wellKnown.RelayServerAddr, "relay.example.com:9123")
		assert.Equal(t, wellKnown.CallerIP, "127.0.0.1")
	})

	t.Run("Connect to websocket", func(t *testing.T) {
		c := &Console{Config: DefaultConfig()}
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
