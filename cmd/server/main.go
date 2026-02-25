package main

import (
	"fmt"
	"os"

	"github.com/aidantrabs/nginx-reload-q/internal/logging"
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

	srv := socket.NewServer(defaultSocketPath, log)

	if err := srv.Listen(); err != nil {
		return err
	}
	defer srv.Close()

	log.Info("ready")

	return srv.Accept()
}
