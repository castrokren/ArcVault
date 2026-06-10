package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

