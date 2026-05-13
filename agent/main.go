package main

import (
"log"
"os"
"runtime"
"time"

"arcvault/agent/heartbeat"
"arcvault/agent/config"
)

func main() {
log.Println("ArcVault Agent starting...")

cfg, err := config.Load("agent-config.yaml")
if err != nil {
log.Fatalf("failed to load config: %v", err)
}

hbCfg := heartbeat.Config{
AgentID:        cfg.AgentID,
CoordinatorURL: cfg.CoordinatorURL,
AuthToken:      cfg.AuthToken,
Interval:       30 * time.Second,
}

// Register with coordinator
hostname, _ := os.Hostname()
if err := heartbeat.Register(hbCfg, hostname, runtime.GOOS, cfg.Version); err != nil {
log.Fatalf("registration failed: %v", err)
}

// Start heartbeat loop (blocking)
heartbeat.Start(hbCfg)
}