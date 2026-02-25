package reloader

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// runs nginx -t then nginx -s reload
func Reload(ctx context.Context) error {
	if err := runCmd(ctx, "nginx", "-t"); err != nil {
		return fmt.Errorf("config test failed: %w", err)
	}

	if err := runCmd(ctx, "nginx", "-s", "reload"); err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}

	return nil
}

func runCmd(ctx context.Context, name string, args ...string) error {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w (%s)", name, args, err, stderr.String())
	}

	return nil
}
