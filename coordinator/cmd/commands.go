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
	"arcvault/coordinator/internal/tlscert"
	"arcvault/coordinator/server"
	"arcvault/coordinator/updater"
	"golang.org/x/crypto/bcrypt"
)

func InitCommand() error {
	fmt.Println("ArcVault Coordinator - Initialization")
	fmt.Println("=====================================")
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter port (default 443): ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	port := 443
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %v", err)
		}
		port = p
	}

	fmt.Print("Enter host (IP or hostname, for TLS cert): ")
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("host is required")
	}

	homeDir, _ := os.UserHomeDir()
	defaultDB := filepath.Join(homeDir, ".arcvault", "arcvault.db")
	fmt.Printf("Enter database path (default %s): ", defaultDB)
	dbPath, _ := reader.ReadString('\n')
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		dbPath = defaultDB
	}

	// Get exe directory for cert files
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	certPath := filepath.Join(exeDir, "cert.pem")
	keyPath := filepath.Join(exeDir, "key.pem")

	// Generate TLS cert
	fmt.Print("\nGenerating TLS certificate...")
	if err := tlscert.Generate(host, certPath, keyPath); err != nil {
		return fmt.Errorf("failed to generate TLS certificate: %v", err)
	}
	fmt.Printf(" done\n")

	// Generate tokens for reference (NOT saved to file)
	adminToken, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %v", err)
	}

	jwtSecret, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %v", err)
	}

	// Save config WITHOUT tokens
	cfg := &config.Config{
		Port:         port,
		DatabasePath: dbPath,
		AdminToken:   "", // Empty — must come from env var
		JWTSecret:    "", // Empty — must come from env var
		Environment:  "development",
		Host:         host,
		CertFile:     certPath,
		KeyFile:      keyPath,
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	configPath, _ := config.GetConfigPath()
	fmt.Printf("\n✓ Configuration saved to: %s\n", configPath)
	fmt.Printf("✓ Database will be initialized at: %s\n", dbPath)
	fmt.Printf("✓ TLS certificate: %s\n\n", certPath)

	// Display setup instructions
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("IMPORTANT: Environment Variables Required")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nBefore running 'coordinator start', set these environment variables:")
	fmt.Printf("\n  export ARCVAULT_ADMIN_TOKEN=%s\n", adminToken)
	fmt.Printf("  export ARCVAULT_JWT_SECRET=%s\n\n", jwtSecret)
	fmt.Println("⚠️  DO NOT share these tokens or commit to git")
	fmt.Println("⚠️  DO NOT restart without exporting these variables")
	fmt.Println("⚠️  For production, use a secret management system (Vault, AWS Secrets Manager, etc.)")
	fmt.Println("\nNext step: Run 'coordinator start'")

	return nil
}

func StartCommand(cfg *config.Config, staticFS fs.FS) error {
	return StartCommandWithContext(cfg, staticFS, nil)
}

// ensureTLSMaterial fills in default cert/key paths next to the executable and
// generates the self-signed pair if it is missing. Idempotent — existing files
// are left untouched — and a no-op when TLS is terminated upstream.
//
// This runs on every start, not just `coordinator init`, because nothing in the
// install path ever called init: the installer writes a config.json with no
// cert_file/key_file, so Server.Start() saw empty paths and quietly fell through
// to ListenAndServe (plain HTTP) while the installer, dashboard and agents all
// addressed it as https://. A fresh install served cleartext on 443.
func ensureTLSMaterial(cfg *config.Config) error {
	if cfg.ExternalTLS {
		return nil
	}

	if cfg.CertFile == "" || cfg.KeyFile == "" {
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not locate executable directory: %w", err)
		}
		exeDir := filepath.Dir(exePath)
		if cfg.CertFile == "" {
			cfg.CertFile = filepath.Join(exeDir, "cert.pem")
		}
		if cfg.KeyFile == "" {
			cfg.KeyFile = filepath.Join(exeDir, "key.pem")
		}
	}

	// Generate() always adds localhost, 127.0.0.1 and os.Hostname() as SANs, so
	// a blank configured host still yields a usable local certificate.
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	return tlscert.EnsureExists(host, cfg.CertFile, cfg.KeyFile)
}

func StartCommandWithContext(cfg *config.Config, staticFS fs.FS, stopCh <-chan struct{}) error {
	log.Printf("Starting ArcVault Coordinator on port %d", cfg.Port)

	if err := ensureTLSMaterial(cfg); err != nil {
		// Serving production traffic in cleartext is worse than not serving:
		// agent tokens and JWTs would go over the wire in the open.
		if cfg.Environment == "production" {
			return fmt.Errorf("could not prepare TLS certificate: %w", err)
		}
		log.Printf("[startup] WARNING: could not prepare TLS certificate (%v) — falling back to plain HTTP", err)
	}

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
		currentVersion = "v0.5.1"
	}

	go startVersionChecker(currentVersion)

	if stopCh == nil {
		return srv.Start()
	}

	// Start server in background, listen for stop signal
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-stopCh:
		log.Println("Shutdown signal received, stopping server...")
		return nil
	}
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

// PruneBootstrapTokensCommand deletes every enrollment (bootstrap) token in
// the database, regardless of expiry, and prints which agent_id hints were
// removed. This is a destructive operator action: any host that has never
// completed its first registration exchange has no other credential and will
// need to be re-enrolled from scratch afterward.
func PruneBootstrapTokensCommand() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.Init(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	hints, err := database.PruneBootstrapTokens()
	if err != nil {
		return fmt.Errorf("failed to prune bootstrap tokens: %w", err)
	}

	if len(hints) == 0 {
		fmt.Println("No bootstrap tokens found — nothing to prune.")
		return nil
	}

	fmt.Printf("Pruned %d bootstrap token(s):\n\n", len(hints))
	for _, h := range hints {
		fmt.Printf("  - %s\n", h)
	}
	fmt.Println("\nAny host among these that has not completed its first registration")
	fmt.Println("exchange has just lost its only credential and must be re-enrolled via")
	fmt.Println("'Enroll Agent' in the dashboard.")
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

// RekeyCertCommand regenerates the TLS certificate.
func RekeyCertCommand() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Host == "" {
		return fmt.Errorf("host not configured (run 'coordinator init' first)")
	}

	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return fmt.Errorf("cert/key paths not configured")
	}

	fmt.Printf("Regenerating TLS certificate for host: %s\n", cfg.Host)
	if err := tlscert.Generate(cfg.Host, cfg.CertFile, cfg.KeyFile); err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	fmt.Printf("Certificate regenerated: %s\n", cfg.CertFile)
	return nil
}
