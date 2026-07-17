package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"arcvault/agent/config"
	"arcvault/agent/heartbeat"
	"arcvault/agent/honcho"
	"arcvault/agent/runner"
	"arcvault/agent/service"
	agentws "arcvault/agent/ws"
)

var Version = "v0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		runAgent()
		return
	}

	switch os.Args[1] {
	case "run-service":
		if err := service.RunService(runAgent); err != nil {
			log.Fatalf("service error: %v", err)
		}
	case "install-service":
		if err := service.Install(); err != nil {
			log.Fatalf("install-service failed: %v", err)
		}
	case "uninstall-service":
		if err := service.Uninstall(); err != nil {
			log.Fatalf("uninstall-service failed: %v", err)
		}
	case "--version", "version":
		fmt.Println(Version)
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runAgent() {
	log.Println("ArcVault Agent starting...")

	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("could not determine executable path: %v", err)
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "agent-config.yaml")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// ── Honcho memory integration ──────────────────────────────────────────
	var hc *honcho.MetricsCollector
	if cfg.HonchoURL != "" {
		honchoClient := honcho.NewClient(cfg.HonchoURL, cfg.AgentID)
		if err := honchoClient.HealthCheck(); err != nil {
			log.Printf("warning: honcho not available at %s: %v (metrics disabled)\n", cfg.HonchoURL, err)
		} else {
			log.Printf("honcho connected at %s (workspace: %s)\n", cfg.HonchoURL, cfg.AgentID)
			hc = honcho.NewMetricsCollector(honchoClient, 22)
			go func() {
				ticker := time.NewTicker(1 * time.Minute)
				defer ticker.Stop()
				for range ticker.C {
					if err := hc.ProcessBatch(); err != nil {
						log.Printf("honcho batch process failed: %v\n", err)
					}
				}
			}()
		}
	} else {
		log.Println("honcho_url not configured — memory metrics disabled")
	}
	// ────────────────────────────────────────────────────────────────────────

	hbCfg := heartbeat.Config{
		AgentID:        cfg.AgentID,
		CoordinatorURL: cfg.CoordinatorURL,
		AuthToken:      cfg.AuthToken,
		CACertFile:     cfg.CACertFile,
		Interval:       30 * time.Second,
	}

	hostname, _ := os.Hostname()
	// Registration must NOT be fatal. On a remote machine the coordinator is
	// often unreachable at service-boot (firewall, boot order, coordinator still
	// starting, TLS pin). A log.Fatalf here exits during StartPending and the SCM
	// reports the opaque "Error 1067: process terminated unexpectedly". Retry in
	// the background until it lands, THEN start heartbeating — heartbeating an
	// agent the coordinator hasn't registered yet just 404s until it catches up.
	go func() {
		for {
			if err := heartbeat.Register(hbCfg, hostname, runtime.GOOS, runtime.GOARCH, Version); err != nil {
				log.Printf("registration failed, retrying in 30s: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}
			break
		}
		heartbeat.Start(hbCfg)
	}()

	r, err := runner.New(runner.Config{
		AgentID:        cfg.AgentID,
		CoordinatorURL: cfg.CoordinatorURL,
		AuthToken:      cfg.AuthToken,
		CACertFile:     cfg.CACertFile,
		PollInterval:   30 * time.Second,
	}, runner.RealExecutor)
	if err != nil {
		log.Fatalf("failed to initialize runner: %v", err)
	}
	if hc != nil {
		r.SetHonchoCollector(hc)
	}
	go r.Start()

	wsClient := &agentws.Client{
		AgentID:        cfg.AgentID,
		CoordinatorURL: cfg.CoordinatorURL,
		Coordinators:   cfg.Coordinators,
		AuthToken:      cfg.AuthToken,
		CACertFile:     cfg.CACertFile,
		Canceller:      r,
		Poller:         r,
	}
	go wsClient.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("ArcVault Agent shutting down...")
	r.Stop()

	if hc != nil {
		log.Println("flushing pending metrics to honcho...")
		if err := hc.Flush(); err != nil {
			log.Printf("honcho flush failed: %v\n", err)
		}
	}
}

func printUsage() {
	fmt.Println("ArcVault Agent")
	fmt.Println("  (no args)          - Run the agent")
	fmt.Println("  install-service    - Install as a system service (requires admin/root)")
	fmt.Println("  uninstall-service  - Remove the system service (requires admin/root)")
	fmt.Println("  help               - Show this help message")
}
