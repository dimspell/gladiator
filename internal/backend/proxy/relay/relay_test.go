package relay_test

import (
	"context"
	"log"

	"github.com/dimspell/gladiator/internal/backend/redirect"
)

// Mocks
type mockRedirect struct {
	id   string
	recv func([]byte) error
}

func (m *mockRedirect) Run(ctx context.Context, handler func([]byte) error) error {
	go func() {
		select {
		case <-ctx.Done():
			return
		}
	}()
	m.recv = handler
	return nil
}

func (m *mockRedirect) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (m *mockRedirect) Close() error {
	log.Printf("Closed redirect: %s", m.id)
	return nil
}

// Mocks for Dial & Listen
func mockDial(id string) func(string, string) (redirect.Redirect, error) {
	return func(ip, port string) (redirect.Redirect, error) {
		return &mockRedirect{id: "dial-" + id}, nil
	}
}
func mockListen(id string) func(string, string) (redirect.Redirect, error) {
	return func(ip, port string) (redirect.Redirect, error) {
		return &mockRedirect{id: "listen-" + id}, nil
	}
}
