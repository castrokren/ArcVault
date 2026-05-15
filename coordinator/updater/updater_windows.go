//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ApplyUpdate stops the service, renames the staged binary, and starts the service.
func ApplyUpdate(stagedPath, currentPath string, progress func(ProgressEvent)) error {
	// Signal restarting
	progress(ProgressEvent{
		Type:    "update_progress",
		Step:    "restarting",
		Pct:     95,
		Message: "Restarting service...",
	})

	// Use Windows Service Control (net stop/start) via cmd.exe
	// Stop service
	stopCmd := exec.Command("net", "stop", "arcvault-coordinator")
	_ = stopCmd.Run() // Ignore error if service not running

	// Give service time to stop
	time.Sleep(500 * time.Millisecond)

	// Rename the staged binary to current
	if err := os.Rename(stagedPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Start service
	startCmd := exec.Command("net", "start", "arcvault-coordinator")
	err := startCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}
