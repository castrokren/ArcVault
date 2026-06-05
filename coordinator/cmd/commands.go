package cmd

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
	"arcvault/coordinator/server"
	"arcvault/coordinator/updater"
	"golang.org/x/crypto/bcrypt"
)

func InitCommand() error {
	fmt.Println("ArcVault Coordinator - Initialization")
	fmt.Println("=====================================")
	reader := bufio.NewReader(os.Stdin)

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

	homeDir, _ := os.UserHomeDir()
	defaultDB := filepath.Join(homeDir, ".arcvault", "arcvault.db")
	fmt.Printf("Enter database path (default %s): ", defaultDB)
	dbPath, _ := reader.ReadString('\n')
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		dbPath = defaultDB
	}

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

func StartCommand(cfg *config.Config, staticFS fs.FS) error {
	log.Printf("Starting ArcVault Coordinator on port %d", cfg.Port)

	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	log.Println("Database initialized")

	// First-run seeding: create default admin user if no users exist
	count, err := database.CountUsers()
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		if err := database.CreateUser("admin", string(hash), "admin", true); err != nil {
			return fmt.Errorf("failed to create default admin user: %w", err)
		}
		log.Println("[startup] Default admin user created (admin/changeme) — change password on first login")
	}

	srv := server.NewWithFS(cfg, database, staticFS)

	// Start background version checker
	currentVersion := os.Getenv("ARCVAULT_VERSION")
	if currentVersion == "" {
		currentVersion = "v0.2.0"
	}

	go startVersionChecker(currentVersion)

	return srv.Start()
}

// startVersionChecker polls GitHub for new releases every 24 hours.
func startVersionChecker(currentVersion string) {
	// Check on startup
	checkAndCache(currentVersion)

	// Check every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		checkAndCache(currentVersion)
	}
}

// checkAndCache fetches the latest release and caches it.
func checkAndCache(currentVersion string) {
	info, err := updater.CheckLatestRelease(currentVersion)
	if err != nil {
		log.Printf("Version check failed (will retry in 24h): %v", err)
		return
	}

	server.SetUpdateCache(info)
	if info.UpdateAvailable {
		log.Printf("New version available: %s (current: %s)", info.Latest, info.Current)
	}
}

// CreateAgentTokenCommand generates a new token for the given agent ID
// and prints it. The token can then be used in agent-config.yaml.
// If tokenOnly is true, prints only the token string (for scripting).
func CreateAgentTokenCommand(agentID string, tokenOnly bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	token, err := database.CreateAgentToken(agentID)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	if tokenOnly {
		fmt.Print(token)
	} else {
		fmt.Printf("Agent token for %q:\n\n  %s\n\n", agentID, token)
		fmt.Println("Add this to agent-config.yaml as auth_token.")
	}
	return nil
}

// CheckUpdateCommand checks for available updates without starting the server.
func CheckUpdateCommand(currentVersion string) error {
	info, err := updater.CheckLatestRelease(currentVersion)
	if err != nil {
		return fmt.Errorf("could not check for updates: %w", err)
	}

	fmt.Printf("current:  %s\n", info.Current)
	fmt.Printf("latest:   %s\n", info.Latest)
	if info.UpdateAvailable {
		fmt.Printf("status:   update available\n")
		fmt.Printf("release:  %s\n", info.ReleaseURL)
	} else {
		fmt.Printf("status:   up to date\n")
	}
	return nil
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

// OpenDatabase opens the coordinator database.
func OpenDatabase(dbPath string) (*db.DB, error) {
	return db.Init(dbPath)
}

// DecodeKeyHex decodes a hex-encoded key and validates it's 32 bytes.
func DecodeKeyHex(keyHex string) ([]byte, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex encoding: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes (got %d)", len(key))
	}
	return key, nil
}
