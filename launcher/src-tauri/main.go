package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	s := http.Server{
		Addr: "127.0.0.1:8081",
	}

	err := graceful(context.Background(),
		func(ctx context.Context) error {
			fmt.Println("Started")
			return s.ListenAndServe()
		},
		func(ctx context.Context) error {
			fmt.Println("Stop")
			return s.Shutdown(ctx)
		},
	)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Done")
}

type gracefulFunc func(context.Context) error

func graceful(ctx context.Context, start gracefulFunc, shutdown gracefulFunc) error {
	var (
		stopChan = make(chan os.Signal, 1)
		errChan  = make(chan error, 1)
	)

	// Set up the graceful shutdown handler (traps SIGINT and SIGTERM)
	go func() {
		signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-stopChan:
			fmt.Println("stopChan")
		case <-ctx.Done():
			fmt.Println("ctx.Done")
		}

		timer, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := shutdown(timer); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	// Start the server
	if err := start(ctx); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-errChan
}
