package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"arcvault/coordinator/cmd"
	"arcvault/coordinator/config"
	"arcvault/coordinator/internal/credcrypto"
	"arcvault/coordinator/service"
	"arcvault/coordinator/static"
)

// Version is injected at build time via ldflags: -X main.Version=vX.Y.Z
// Fallback must match VERSION file at repo root — update both together.
var Version = "v0.5.1"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--version", "version":
		fmt.Println(Version)
	case "init":
		if err := cmd.InitCommand(); err != nil {
			log.Fatalf("init failed: %v", err)
		}
	case "start":
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		// Set version in environment for update checks
		os.Setenv("ARCVAULT_VERSION", Version)
		if err := cmd.StartCommand(cfg, static.FS()); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case "create-agent-token":
		if len(os.Args) < 3 {
			fmt.Println("Usage: coordinator create-agent-token <agent-id> [--token-only]")
			os.Exit(1)
		}
		tokenOnly := len(os.Args) > 3 && os.Args[3] == "--token-only"
		if err := cmd.CreateAgentTokenCommand(os.Args[2], tokenOnly); err != nil {
			log.Fatalf("create-agent-token failed: %v", err)
		}
	case "rekey":
		if len(os.Args) < 5 {
			fmt.Println("Usage: coordinator rekey --old-key <hex> --new-key <hex>")
			os.Exit(1)
		}

		var oldKey, newKey string
		for i := 2; i < len(os.Args)-1; i++ {
			if os.Args[i] == "--old-key" && i+1 < len(os.Args) {
				oldKey = os.Args[i+1]
			}
			if os.Args[i] == "--new-key" && i+1 < len(os.Args) {
				newKey = os.Args[i+1]
			}
		}

		if oldKey == "" || newKey == "" {
			fmt.Println("Usage: coordinator rekey --old-key <hex> --new-key <hex>")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}

		database, err := cmd.OpenDatabase(cfg.DatabasePath)
		if err != nil {
			log.Fatalf("failed to open database: %v", err)
		}
		defer database.Close()

		oldKeyBytes, err := cmd.DecodeKeyHex(oldKey)
		if err != nil {
			log.Fatalf("invalid old-key: %v", err)
		}

		newKeyBytes, err := cmd.DecodeKeyHex(newKey)
		if err != nil {
			log.Fatalf("invalid new-key: %v", err)
		}

		if err := credcrypto.Rekey(database.Conn(), oldKeyBytes, newKeyBytes); err != nil {
			log.Fatalf("rekey failed: %v", err)
		}

		fmt.Println("Credentials rekeyed successfully")
		os.Exit(0)
	case "rekey-cert":
		if err := cmd.RekeyCertCommand(); err != nil {
			log.Fatalf("rekey-cert failed: %v", err)
		}
		fmt.Println("TLS certificate regenerated successfully")
		os.Exit(0)
	case "check-update":
		if err := cmd.CheckUpdateCommand(Version); err != nil {
			log.Fatalf("check-update failed: %v", err)
		}
	case "run-service":
		// Called by Windows SCM — wraps StartCommand inside svc.Run() so the
		// Service Control Manager gets the handshake it requires.
		redirectServiceLog("coordinator-service.log")
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		os.Setenv("ARCVAULT_VERSION", Version)
		if err := service.RunService(cfg, static.FS()); err != nil {
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
	case "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// redirectServiceLog points the standard logger at a file beside the exe.
// Under the Windows SCM there is no console, so a startup failure (missing
// config, port 443 in use, TLS error) otherwise dies as an opaque
// "Error 1067: process terminated unexpectedly" with no trace.
func redirectServiceLog(name string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(filepath.Dir(exe), name),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	log.SetOutput(f)
}

func printUsage() {
	fmt.Println("ArcVault Coordinator")
	fmt.Println("  init                          - Initialize and generate admin token")
	fmt.Println("  start                         - Start the coordinator server")
	fmt.Println("  create-agent-token <agent-id> - Generate a token for an agent")
	fmt.Println("  rekey <args>                  - Rotate credential encryption keys")
	fmt.Println("  rekey-cert                    - Regenerate TLS certificate")
	fmt.Println("  check-update                  - Check for available updates")
	fmt.Println("  install-service               - Install as a system service (requires admin/root)")
	fmt.Println("  uninstall-service             - Remove the system service (requires admin/root)")
	fmt.Println("  help                          - Show this help message")
}
