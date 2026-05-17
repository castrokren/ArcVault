package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
AgentID        string
CoordinatorURL string
AuthToken      string
Interval       time.Duration
}

type heartbeatResponse struct {
Status string `json:"status"`
Time   string `json:"time"`
}

func Start(cfg Config) {
if cfg.Interval == 0 {
cfg.Interval = 30 * time.Second
}

log.Printf("Heartbeat loop started (every %s)", cfg.Interval)

for {
if err := send(cfg); err != nil {
log.Printf("Heartbeat failed: %v", err)
}
time.Sleep(cfg.Interval)
}
}

func Register(cfg Config, hostname, os, arch, version string) error {
body, _ := json.Marshal(map[string]string{
"agent_id": cfg.AgentID,
"hostname": hostname,
"os":       os,
"arch":     arch,
"version":  version,
})

req, err := http.NewRequest("POST", cfg.CoordinatorURL+"/api/agents/register", bytes.NewBuffer(body))
if err != nil {
return err
}
req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
req.Header.Set("Content-Type", "application/json")

resp, err := http.DefaultClient.Do(req)
if err != nil {
return fmt.Errorf("registration request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusCreated {
return fmt.Errorf("registration failed with status %d", resp.StatusCode)
}

log.Printf("Registered with coordinator as %s", cfg.AgentID)
return nil
}

func send(cfg Config) error {
	// Check if rollback is available (backup binary exists)
	rollbackAvailable := isRollbackAvailable()

	// Build heartbeat payload
	payload := map[string]bool{
		"rollback_available": rollbackAvailable,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/agents/%s/heartbeat", cfg.CoordinatorURL, cfg.AgentID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var hbResp heartbeatResponse
	json.NewDecoder(resp.Body).Decode(&hbResp)
	log.Printf("Heartbeat OK at %s", hbResp.Time)
	return nil
}

// isRollbackAvailable checks if a backup binary exists for rollback.
func isRollbackAvailable() bool {
	var backupDir string
	switch os.Getenv("GOOS") {
	case "windows", "":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = filepath.Join(os.Getenv("SystemDrive"), "ProgramData")
		}
		backupDir = filepath.Join(programData, "ArcVault", "backups")
	default: // linux, darwin
		backupDir = "/var/lib/arcvault/backups"
	}

	backupPath := filepath.Join(backupDir, "agent.previous")
	_, err := os.Stat(backupPath)
	return err == nil
}