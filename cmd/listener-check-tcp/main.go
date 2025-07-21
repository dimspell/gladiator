package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/probe"
)

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	host, port := "127.0.0.1", "21370"

	l, err := redirect.NewListenerTCP(host, port, func(p []byte) (err error) {
		log.Printf("Received on TCP %s", p)
		return nil
	})
	if err != nil {
		log.Fatalf("listener start error: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Fatalf("close error: %v", err)
			return
		}
	}()

	// lctx, cancel := context.WithCancel(context.Background())

	lctx := context.Background()

	// go func() {
	// 	time.Sleep(3 * time.Second)
	// 	cancel()
	// }()

	go func() {
		if err := l.Run(lctx); err != nil {
			log.Printf("run error: %v", err)
			return
		}
	}()

	errProbe := probe.StartProbeTCP(context.Background(), net.JoinHostPort(host, port), func() {
		log.Println("Closing probe 1....")
	})
	if errProbe != nil {
		log.Fatalf("probe error: %v", err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		ticker := time.NewTicker(2 * time.Second)

		for now := range ticker.C {
			fmt.Println("Alive", l.Alive(now, 5*time.Second), now.Format(time.TimeOnly))
		}
	}()

	time.Sleep(100 * time.Second)

	// errProbe2 := probe.StartProbeTCP(context.Background(), net.JoinHostPort(host, port), func() {
	// 	log.Println("Closing probe 2....")
	// })
	// if errProbe2 != nil {
	// 	log.Fatalf("probe2 error: %v", err)
	// }

	select {}
}
