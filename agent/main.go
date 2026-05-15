package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"arcvault/agent/config"
	"arcvault/agent/heartbeat"
	"arcvault/agent/runner"
	"arcvault/agent/service"
)

func main() {
	// no args = run the agent (backward compatible)
	if len(os.Args) < 2 {
		runAgent()
		return
	}

	switch os.Args[1] {
	case "install-service":
		if err := service.Install(); err != nil {
			log.Fatalf("install-service failed: %v", err)
		}
	case "uninstall-service":
		if err := service.Uninstall(); err != nil {
			log.Fatalf("uninstall-service failed: %v", err)
		}
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

func printUsage() {
	fmt.Println("ArcVault Agent")
	fmt.Println("  (no args)          - Run the agent")
	fmt.Println("  install-service    - Install as a system service (requires admin/root)")
	fmt.Println("  uninstall-service  - Remove the system service (requires admin/root)")
	fmt.Println("  help               - Show this help message")
}
