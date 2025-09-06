//go:build !windows

package par2

import (
	"context"
	"os/exec"
)

// createCommand creates a standard command context for non-Windows platforms
func createCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}