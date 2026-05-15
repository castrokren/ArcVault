package main

import (
	"fmt"
	"log"
	"os"

	"arcvault/coordinator/cmd"
	"arcvault/coordinator/config"
	"arcvault/coordinator/service"
	"arcvault/coordinator/static"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
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

func printUsage() {
	fmt.Println("ArcVault Coordinator")
	fmt.Println("  init               - Initialize coordinator and generate admin token")
	fmt.Println("  start              - Start the coordinator server")
	fmt.Println("  install-service    - Install as a system service (requires admin/root)")
	fmt.Println("  uninstall-service  - Remove the system service (requires admin/root)")
	fmt.Println("  help               - Show this help message")
}
