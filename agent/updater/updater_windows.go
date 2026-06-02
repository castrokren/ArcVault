//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ApplyUpdate stops the agent service, replaces the binary, and starts the service.
func ApplyUpdate(stagedPath, currentPath string) error {
	if err := exec.Command("net", "stop", "arcvault-agent").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	if err := os.Rename(stagedPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	if err := exec.Command("net", "start", "arcvault-agent").Run(); err != nil {
		return fmt.Errorf("service started failed after binary replacement: %w", err)
	}
	return nil
}
