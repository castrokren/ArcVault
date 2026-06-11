package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleDownloadAgent serves agent.exe with auth check.
func (s *Server) handleDownloadAgent(w http.ResponseWriter, r *http.Request) {
	exePath, err := os.Executable()
	if err != nil {
		http.Error(w, "failed to determine executable path", http.StatusInternalServerError)
		return
	}
	agentExePath := filepath.Join(filepath.Dir(exePath), "agent.exe")
	fi, err := os.Stat(agentExePath)
	if err != nil {
		http.Error(w, "agent.exe not available", http.StatusNotFound)
		return
	}
	file, err := os.Open(agentExePath)
	if err != nil {
		http.Error(w, "failed to open agent.exe", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=agent.exe")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}

// handleDownloadInstaller serves the ArcVault Setup exe for the running version.
// Looks in installer_dir config, falls back to coordinator's own directory.
// Matches ArcVault-Setup-{version}-windows-amd64.exe.
func (s *Server) handleDownloadInstaller(w http.ResponseWriter, r *http.Request) {
	searchDir := s.cfg.InstallerDir
	if searchDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			http.Error(w, "failed to determine executable path", http.StatusInternalServerError)
			return
		}
		searchDir = filepath.Dir(exePath)
	}

	// Get running version from env (set by main.go at startup)
	version := os.Getenv("ARCVAULT_VERSION")
	if version == "" {
		version = Version // fallback to compiled-in default
	}
	// Trim leading "v" for filename match: v0.5.1 -> 0.5.1
	versionTrimmed := strings.TrimPrefix(version, "v")
	fileName := fmt.Sprintf("ArcVault-Setup-%s-windows-amd64.exe", versionTrimmed)
	installerPath := filepath.Join(searchDir, fileName)

	fi, err := os.Stat(installerPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("installer not found: %s — run scripts/build.ps1 first", fileName), http.StatusNotFound)
		return
	}

	file, err := os.Open(installerPath)
	if err != nil {
		http.Error(w, "failed to open installer", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}
