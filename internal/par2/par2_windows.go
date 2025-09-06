//go:build windows

package par2

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// createCommand creates a command context with Windows-specific attributes to hide the console window
func createCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	
	// Hide the console window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	
	return cmd
}