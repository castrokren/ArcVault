package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"arcvault/coordinator/updater"
)

var (
	agentUpdateMu      sync.Mutex
	agentUpdatesInProgress = make(map[string]bool)
)

type updateCommandMsg struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

func (s *Server) handleAgentUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	// Guard against concurrent updates for the same agent.
	agentUpdateMu.Lock()
	if agentUpdatesInProgress[agentID] {
		agentUpdateMu.Unlock()
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "update already in progress for this agent"})
		return
	}
	agentUpdatesInProgress[agentID] = true
	agentUpdateMu.Unlock()

	defer func() {
		agentUpdateMu.Lock()
		delete(agentUpdatesInProgress, agentID)
		agentUpdateMu.Unlock()
	}()

	// Look up agent OS/arch from DB.
	var goos, goarch string
	err := s.db.Conn().QueryRow(
		`SELECT os, arch FROM agents WHERE id = ?`, agentID,
	).Scan(&goos, &goarch)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if goarch == "" {
		http.Error(w, "agent arch unknown — agent must re-register", http.StatusBadRequest)
		return
	}

	// Fetch latest release assets.
	assets, _, err := updater.FetchLatestRelease()
	if err != nil {
		log.Printf("agent update: failed to fetch release: %v", err)
		http.Error(w, fmt.Sprintf("failed to fetch release info: %v", err), http.StatusInternalServerError)
		return
	}

	assetURL, err := updater.ResolveAgentAssetURL(goos, goarch, assets)
	if err != nil {
		http.Error(w, fmt.Sprintf("no asset for agent platform %s/%s: %v", goos, goarch, err), http.StatusBadRequest)
		return
	}

	// Determine target version from cached update info (coordinator shares the same release).
	info := GetUpdateCache()
	targetVersion := ""
	if info != nil {
		targetVersion = info.Latest
	}

	cmd := updateCommandMsg{
		Type:    "update_command",
		Version: targetVersion,
		URL:     assetURL,
	}

	if err := s.hub.SendToAgent(agentID, cmd); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("agent not connected: %v", err)})
		return
	}

	log.Printf("agent update: sent update_command to %s (version=%s)", agentID, targetVersion)
	json.NewEncoder(w).Encode(map[string]string{"status": "update_command sent", "agent_id": agentID})
}
