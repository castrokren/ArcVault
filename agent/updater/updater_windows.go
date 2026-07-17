//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ApplyUpdate defers the binary swap + service restart to a detached helper process,
// then stops the agent service. The detached helper polls for the service stop,
// swaps the binary, and restarts the service — all after the agent process exits.
// This prevents the self-kill race where the service SCM handler calls os.Exit(0)
// while we're still inside exec.Command("net", "stop", ...).Run(), blocking on
// the stop command.
func ApplyUpdate(stagedPath, currentPath string) error {
	// Get the backup directory so we can write the helper script next to it
	backupDir, err := getBackupDir()
	if err != nil {
		return fmt.Errorf("failed to get backup directory: %w", err)
	}

	helperPath := filepath.Join(backupDir, "arcvault-update-helper.cmd")

	// Generate and write the helper script to disk.
	// The script will poll for the service stop, move the binary, and restart the service.
	helperScript := fmt.Sprintf(`@echo off
REM ArcVault agent update helper — runs after the service stops.
REM Polls for the service to reach STOPPED state, swaps the binary, and restarts.

setlocal enabledelayedexpansion
set STAGED=%s
set CURRENT=%s
set MAX_POLLS=60
set POLL_COUNT=0

:poll_loop
sc query arcvault-agent | find "STOPPED" >nul
if errorlevel 1 (
    set /a POLL_COUNT=!POLL_COUNT!+1
    if !POLL_COUNT! geq !MAX_POLLS! (
        echo Agent service did not stop within 60 seconds. Giving up.
        exit /b 1
    )
    timeout /t 1 /nobreak >nul
    goto poll_loop
)

REM Service is stopped. Swap the binary.
move /y "!STAGED!" "!CURRENT!" >nul 2>&1
if errorlevel 1 (
    echo Failed to replace binary.
    exit /b 1
)

REM Restart the service.
net start arcvault-agent >nul 2>&1
if errorlevel 1 (
    echo Failed to start service after binary replacement.
    exit /b 1
)

exit /b 0
`, stagedPath, currentPath)

	if err := os.WriteFile(helperPath, []byte(helperScript), 0644); err != nil {
		return fmt.Errorf("failed to write helper script: %w", err)
	}

	// Launch the helper script as a detached process so it survives our exit.
	// Use cmd /c start /min /b to launch the helper with minimal window, and
	// apply the DETACHED_PROCESS flag via SysProcAttr.
	// This ensures the helper continues running even after the agent process is terminated.
	cmd := exec.Command("cmd", "/c", "start", "/min", "/b", helperPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008, // DETACHED_PROCESS = 0x00000008
	}

	// Fire-and-forget: we don't wait for the helper to complete.
	// It will poll, swap, and restart while we proceed to exit.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update helper: %w", err)
	}

	// Now stop the agent service. The helper is already running detached and will
	// take over once the service is stopped. We don't need to swap the binary
	// or restart here — the helper owns that.
	if err := exec.Command("net", "stop", "arcvault-agent").Run(); err != nil {
		// Log the error but don't fail — the helper is still running and may recover.
		// The service may already be stopping or stopped.
		fmt.Printf("net stop exited with error (helper is running): %v\n", err)
	}

	return nil
}
