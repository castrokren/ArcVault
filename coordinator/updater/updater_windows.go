//go:build windows

package updater

import (
	"fmt"
	"os"
	"time"
)

// ApplyUpdate replaces the binary in place and exits so the SCM restarts the service.
func ApplyUpdate(stagedPath, currentPath string, progress func(ProgressEvent)) error {
	progress(ProgressEvent{
		Type:    "update_progress",
		Step:    "restarting",
		Pct:     95,
		Message: "Restarting service...",
	})

	// Rename current binary out of the way
	oldPath := currentPath + ".old"
	_ = os.Remove(oldPath)

	var renameErr error
	for i := 0; i < 10; i++ {
		if err := os.Rename(currentPath, oldPath); err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		if err := os.Rename(stagedPath, currentPath); err != nil {
			_ = os.Rename(oldPath, currentPath)
			renameErr = err
			break
		}
		renameErr = nil
		break
	}
	if renameErr != nil {
		return fmt.Errorf("failed to replace binary: %w", renameErr)
	}

	// Exit so the SCM failure recovery restarts the service with the new binary
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(1)
	}()

	return nil
}
