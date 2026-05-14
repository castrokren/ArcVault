package main

import (
	"fmt"
	"log"
	"os"

	"arcvault/coordinator/cmd"
	"arcvault/coordinator/config"
	"arcvault/coordinator/static"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: coordinator init|start|help")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := cmd.InitCommand(); err != nil {
			log.Fatalf("init failed: %v", err)
		}
	case "start":
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		if err := cmd.StartCommand(cfg, static.FS()); err != nil {
			log.Fatalf("server error: %v", err)
		}
	case "help":
		fmt.Println("ArcVault Coordinator")
		fmt.Println("  init   - Initialize coordinator and generate admin token")
		fmt.Println("  start  - Start the coordinator server")
		fmt.Println("  help   - Show this help message")
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: coordinator init|start|help")
		os.Exit(1)
	}
}
