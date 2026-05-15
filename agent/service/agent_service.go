package service

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	AgentServiceName = "arcvault-agent"
	AgentDisplayName = "ArcVault Agent"
	AgentDescription = "ArcVault backup agent service"
)

// Install registers the agent as a system service.
// Must be run with administrator/root privileges.
func Install() error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	return install(exe)
}

// Uninstall removes the agent system service.
// Must be run with administrator/root privileges.
func Uninstall() error {
	return uninstall()
}

func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not determine executable path: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}
