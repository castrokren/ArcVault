//go:build darwin

package updater

import (
	"fmt"
	"os"
	"os/exec"
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

	// Stop launchctl service
	if err := exec.Command("launchctl", "stop", "com.arcvault.coordinator").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Rename the staged binary to current
	if err := os.Rename(stagedPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Start launchctl service
	if err := exec.Command("launchctl", "start", "com.arcvault.coordinator").Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// The coordinator will be killed by launchctl stop, and a new one starts
	return nil
}
