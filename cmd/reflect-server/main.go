package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/dimspell/gladiator/internal/relayserver"
)

func main() {
	addr := flag.String("addr", ":9978", "QUIC relay server address")
	flag.Usage = func() {
		log.Printf("Usage: reflect-server [--addr <host:port>]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	rs, err := relayserver.NewRelayServer(*addr, relayserver.AllowAllSessions())
	if err != nil {
		log.Fatalf("Failed to start relay server: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("Reflect relay server starting", "addr", rs.Addr())
	go rs.Start(ctx)

	<-ctx.Done()
	slog.Info("Shutting down relay server")
}
