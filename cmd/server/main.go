package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aidantrabs/nginx-reload-q/internal/logging"
	"github.com/aidantrabs/nginx-reload-q/internal/queue"
	"github.com/aidantrabs/nginx-reload-q/internal/reloader"
	"github.com/aidantrabs/nginx-reload-q/internal/socket"
)

func main() {
	fmt.Println("starting")

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

const defaultSocketPath = "/var/run/nginx-reload.sock"

func run() error {
	log := logging.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := queue.New(16, reloader.Reload, log)
	q.Start(ctx)
	defer q.Close()

	srv := socket.NewServer(defaultSocketPath, q, log)

	if err := srv.Listen(); err != nil {
		return err
	}
	defer srv.Close()

	log.Info("ready")

	// shut down cleanly on SIGTERM or SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Accept()
	}()

	select {
	case err := <-errCh:
		return err
	case s := <-sig:
		log.Info("shutting down", "signal", s.String())
		cancel()
		return nil
	}
}
