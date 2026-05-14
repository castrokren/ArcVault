package cmd

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
	"arcvault/coordinator/server"
)

func InitCommand() error {
	fmt.Println("ArcVault Coordinator - Initialization")
	fmt.Println("=====================================")
	reader := bufio.NewReader(os.Stdin)

	// Get port
	fmt.Print("Enter port (default 8080): ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	port := 8080
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %v", err)
		}
		port = p
	}

	// Get database path
	homeDir, _ := os.UserHomeDir()
	defaultDB := filepath.Join(homeDir, ".arcvault", "arcvault.db")
	fmt.Printf("Enter database path (default %s): ", defaultDB)
	dbPath, _ := reader.ReadString('\n')
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		dbPath = defaultDB
	}

	// Generate admin token
	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %v", err)
	}

	cfg := &config.Config{
		Port:         port,
		DatabasePath: dbPath,
		AdminToken:   token,
		Environment:  "development",
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\nConfiguration saved to: %s\n", configPath)
	fmt.Printf("Database will be initialized at: %s\n", dbPath)
	fmt.Printf("Admin token (save this): %s\n\n", token)
	fmt.Println("Next step: Run 'coordinator start'")
	return nil
}

// StartCommand starts the coordinator. staticFS is the embedded dashboard
// filesystem — pass nil to skip static serving (e.g. in dev without a build).
func StartCommand(cfg *config.Config, staticFS fs.FS) error {
	log.Printf("Starting ArcVault Coordinator on port %d", cfg.Port)

	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("Database initialized")

	srv := server.NewWithFS(cfg, database, staticFS)
	return srv.Start()
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hexStr := ""
	for _, b := range bytes {
		hexStr += fmt.Sprintf("%02x", b)
	}
	return hexStr, nil
}
