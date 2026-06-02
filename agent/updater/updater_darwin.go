//go:build darwin

package updater

import (
	"fmt"
	"os"
	"os/exec"
)

// ApplyUpdate stops the agent service, replaces the binary, and starts the service.
func ApplyUpdate(stagedPath, currentPath string) error {
	if err := exec.Command("launchctl", "stop", "com.arcvault.agent").Run(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	if err := os.Rename(stagedPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	if err := exec.Command("launchctl", "start", "com.arcvault.agent").Run(); err != nil {
		return fmt.Errorf("service start failed after binary replacement: %w", err)
	}
	return nil
}
