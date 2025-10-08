package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"golang.org/x/sync/errgroup"
)

type mockWriter struct {
	onWrite func(p []byte) (int, error)
}

func (m *mockWriter) Write(p []byte) (int, error) {
	if m.onWrite == nil {
		log.Printf("wrote to dc: %s\n", string(p))
		return len(p), nil
	}
	return m.onWrite(p)
}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx := context.Background()

	listenerTCP, err := redirect.NewListenerTCP("127.0.0.1", "61140", nil)
	if err != nil {
		log.Fatal(err)
	}

	listenerUDP, err := redirect.NewListenerUDP("127.0.0.1", "61130", nil)
	if err != nil {
		log.Fatal(err)
	}

	dialTCP, err := redirect.NewDialTCP("127.0.0.1", "6114", nil)
	if err != nil {
		log.Fatal(err)
	}

	dialUDP, err := redirect.NewDialUDP("127.0.0.1", "6113", nil)
	if err != nil {
		log.Fatal(err)
	}

	redirectTCP := &mockWriter{
		onWrite: func(p []byte) (int, error) {
			return dialTCP.Write(p)
		},
	}
	redirectUDP := &mockWriter{
		onWrite: func(p []byte) (int, error) {
			return dialUDP.Write(p)
		},
	}

	listenerTCP.OnReceive = func(p []byte) (err error) {
		_, err = redirectTCP.Write(p)
		return err
	}
	listenerUDP.OnReceive = func(p []byte) (err error) {
		_, err = redirectUDP.Write(p)
		return err
	}
	dialTCP.OnReceive = func(p []byte) (err error) {
		_, err = listenerTCP.Write(p)
		return err
	}
	dialUDP.OnReceive = func(p []byte) (err error) {
		_, err = listenerUDP.Write(p)
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return listenerTCP.Run(ctx)
	})
	g.Go(func() error {
		return listenerUDP.Run(ctx)
	})
	g.Go(func() error {
		return dialTCP.Run(ctx)
	})
	g.Go(func() error {
		return dialUDP.Run(ctx)
	})
	if err := g.Wait(); err != nil {
		log.Println(err)
		return
	}
}
