package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// handleDownloadAgent serves agent.exe with auth check.
// Accepts: agent token OR admin token
func (s *Server) handleDownloadAgent(w http.ResponseWriter, r *http.Request) {
	// Get executable path and look for agent.exe in the same directory
	exePath, err := os.Executable()
	if err != nil {
		http.Error(w, "failed to determine executable path", http.StatusInternalServerError)
		return
	}

	// Replace coordinator.exe with agent.exe
	agentExePath := filepath.Join(filepath.Dir(exePath), "agent.exe")

	// Check if file exists
	fi, err := os.Stat(agentExePath)
	if err != nil {
		http.Error(w, "agent.exe not available", http.StatusNotFound)
		return
	}

	// Open the file
	file, err := os.Open(agentExePath)
	if err != nil {
		http.Error(w, "failed to open agent.exe", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set response headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=agent.exe")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))

	// Stream the file
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}

// handleDownloadInstaller serves the ArcVault Setup .exe from the configured installer_dir.
// Falls back to the coordinator's own directory if installer_dir is not set.
// Admin-only endpoint.
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

	// Find all ArcVault-Setup-*.exe files in the directory
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("installer directory not accessible: %v", err), http.StatusNotFound)
		return
	}

	var candidates []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "ArcVault-Setup-") && strings.HasSuffix(e.Name(), ".exe") {
			candidates = append(candidates, e)
		}
	}

	if len(candidates) == 0 {
		http.Error(w, "no installer found — build one first with scripts/build.ps1", http.StatusNotFound)
		return
	}

	// Pick the newest by file modification time
	sort.Slice(candidates, func(i, j int) bool {
		ii, _ := candidates[i].Info()
		jj, _ := candidates[j].Info()
		if ii == nil || jj == nil {
			return false
		}
		return ii.ModTime().After(jj.ModTime())
	})

	installerPath := filepath.Join(searchDir, candidates[0].Name())
	fi, err := os.Stat(installerPath)
	if err != nil {
		http.Error(w, "failed to stat installer", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(installerPath)
	if err != nil {
		http.Error(w, "failed to open installer", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", candidates[0].Name()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}
