package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"arcvault/coordinator/updater"
)

var (
	updateMu      sync.RWMutex
	cachedInfo    *updater.UpdateInfo
	updateRunning atomic.Bool
)

// SetUpdateCache stores the latest version check result.
func SetUpdateCache(info *updater.UpdateInfo) {
	updateMu.Lock()
	defer updateMu.Unlock()
	cachedInfo = info
}

// GetUpdateCache returns the cached version check result.
func GetUpdateCache() *updater.UpdateInfo {
	updateMu.RLock()
	defer updateMu.RUnlock()
	return cachedInfo
}

// handleCheckUpdate returns the cached update information.
func (s *Server) handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := GetUpdateCache()
	if info == nil {
		// If no cache yet, do an inline check
		// Try to get version from environment or default
		currentVersion := os.Getenv("ARCVAULT_VERSION")
		if currentVersion == "" {
			currentVersion = "v0.2.0"
		}

		var err error
		info, err = updater.CheckLatestRelease(currentVersion)
		if err != nil {
			log.Printf("Failed to check latest release: %v", err)
			http.Error(w, fmt.Sprintf("failed to check updates: %v", err), http.StatusInternalServerError)
			return
		}

		SetUpdateCache(info)
	}

	json.NewEncoder(w).Encode(info)
}

// handleApplyUpdate starts the update process.
func (s *Server) handleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if update is already running
	if updateRunning.Load() {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "update already in progress",
		})
		return
	}

	// Set running flag
	updateRunning.Store(true)
	defer updateRunning.Store(false)

	// Get current version
	currentVersion := os.Getenv("ARCVAULT_VERSION")
	if currentVersion == "" {
		currentVersion = "v0.2.0"
	}

	// Start update in goroutine
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "update started",
	})

	go func() {
		if err := performUpdate(currentVersion, s); err != nil {
			log.Printf("Update failed: %v", err)
		}
	}()
}

// performUpdate executes the full update flow.
func performUpdate(currentVersion string, s *Server) error {
	progress := func(evt updater.ProgressEvent) {
		s.hub.Broadcast(Event{
			Type:    "update_progress",
			Payload: evt,
		})
	}

	// Check latest release
	progress(updater.ProgressEvent{
		Type:    "update_progress",
		Step:    "resolving",
		Pct:     10,
		Message: "Resolving release asset...",
	})

	info, err := updater.CheckLatestRelease(currentVersion)
	if err != nil {
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("failed to check release: %s", err.Error()),
		})
		return err
	}

	// Download binary
	progress(updater.ProgressEvent{
		Type:    "update_progress",
		Step:    "downloading",
		Pct:     20,
		Message: fmt.Sprintf("Downloading coordinator binary..."),
	})

	exePath, err := os.Executable()
	if err != nil {
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("could not determine executable path: %s", err.Error()),
		})
		return err
	}

	exeDir := filepath.Dir(exePath)
	tmpPath := filepath.Join(exeDir, "coordinator.download.tmp")
	stagedPath := filepath.Join(exeDir, "coordinator.new")

	downloadProgress := func(pct int) {
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "downloading",
			Pct:     20 + (pct * 35 / 100), // Scale to 20-55%
			Message: fmt.Sprintf("Downloading... %d%%", pct),
		})
	}

	if err := updater.DownloadBinary(info.AssetURL, tmpPath, downloadProgress); err != nil {
		os.Remove(tmpPath)
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("download failed: %s", err.Error()),
		})
		return err
	}

	// Verify binary
	progress(updater.ProgressEvent{
		Type:    "update_progress",
		Step:    "verifying",
		Pct:     80,
		Message: "Verifying binary...",
	})

	if err := updater.VerifyBinary(tmpPath); err != nil {
		os.Remove(tmpPath)
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("binary verification failed: %s", err.Error()),
		})
		return err
	}

	// Stage binary
	progress(updater.ProgressEvent{
		Type:    "update_progress",
		Step:    "staging",
		Pct:     88,
		Message: "Staging binary...",
	})

	if err := updater.StageBinary(tmpPath, stagedPath); err != nil {
		os.Remove(tmpPath)
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("failed to stage binary: %s", err.Error()),
		})
		return err
	}

	// Apply update (platform-specific)
	if err := updater.ExecuteUpdate(stagedPath, exePath, progress); err != nil {
		// Clean up staged file
		os.Remove(stagedPath)
		progress(updater.ProgressEvent{
			Type:    "update_progress",
			Step:    "error",
			Pct:     -1,
			Message: fmt.Sprintf("failed to apply update: %s", err.Error()),
		})
		return err
	}

	// Terminal mode emits done_manual, service mode will restart
	if !updater.IsServiceMode() {
		// Already handled in ExecuteUpdate
		return nil
	}

	// Service mode: emit done event
	progress(updater.ProgressEvent{
		Type:    "update_progress",
		Step:    "done",
		Pct:     100,
		Message: "Restarted. Reconnecting...",
	})

	return nil
}
