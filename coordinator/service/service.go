package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	CoordinatorServiceName = "arcvault-coordinator"
	CoordinatorDisplayName = "ArcVault Coordinator"
	CoordinatorDescription = "ArcVault backup coordinator service"
)

// Install registers the coordinator as a system service.
// Must be run with administrator/root privileges.
func Install() error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	return install(exe)
}

// Uninstall removes the coordinator system service.
// Must be run with administrator/root privileges.
func Uninstall() error {
	return uninstall()
}

// executablePath returns the absolute path of the running executable.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}

// isCommandAvailable checks if a command exists on PATH.
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
