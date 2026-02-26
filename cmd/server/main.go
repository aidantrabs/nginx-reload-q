package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/aidantrabs/nginx-reload-q/internal/logging"
	"github.com/aidantrabs/nginx-reload-q/internal/metrics"
	"github.com/aidantrabs/nginx-reload-q/internal/queue"
	"github.com/aidantrabs/nginx-reload-q/internal/reloader"
	"github.com/aidantrabs/nginx-reload-q/internal/socket"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}

	fmt.Println("starting")

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func run() error {
	socketPath := flag.String("socket", envOrDefault("RELOAD_SOCKET", "/var/run/nginx-reload.sock"), "path to unix socket")
	metricsAddr := flag.String("metrics", envOrDefault("RELOAD_METRICS_ADDR", "127.0.0.1:9111"), "metrics listen address")
	flag.Parse()

	log := logging.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := queue.New(reloader.Reload, log)
	q.Start(ctx)

	srv := socket.NewServer(*socketPath, q, log)

	if err := srv.Listen(); err != nil {
		return err
	}

	msrv := metrics.NewServer(*metricsAddr, q, log)

	log.Info("ready")

	// shut down cleanly on SIGTERM or SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Accept()
	}()
	go func() {
		if err := msrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case s := <-sig:
		log.Info("shutting down", "signal", s.String())
	}

	// shutdown order matters:
	// 1. stop accepting new connections
	// 2. cancel context so in-flight reloads abort
	// 3. drain the queue and wait for worker to finish
	// 4. stop metrics server
	srv.Close()
	cancel()
	q.Close()
	msrv.Close()

	log.Info("shutdown complete")

	return nil
}
