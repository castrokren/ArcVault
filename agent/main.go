package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"arcvault/agent/config"
	"arcvault/agent/heartbeat"
	"arcvault/agent/runner"
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

	// Start heartbeat loop in background
	go heartbeat.Start(hbCfg)

	// Start job runner in background
	r := runner.New(runner.Config{
		AgentID:        cfg.AgentID,
		CoordinatorURL: cfg.CoordinatorURL,
		AuthToken:      cfg.AuthToken,
		PollInterval:   30 * time.Second,
	}, runner.RealExecutor)
	go r.Start()

	// Block until CTRL+C or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("ArcVault Agent shutting down...")
	r.Stop()
}
