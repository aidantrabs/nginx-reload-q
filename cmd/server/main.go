package main

import (
	"fmt"
	"os"

	"github.com/aidantrabs/nginx-reload-q/internal/logging"
)

func main() {
	fmt.Println("starting")

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	log := logging.New()

	log.Info("ready")

	return nil
}
