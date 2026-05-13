package cmd

import (
"bufio"
"crypto/rand"
"fmt"
"log"
"os"
"strconv"
"strings"

"arcvault/coordinator/config"
"arcvault/coordinator/db"
"arcvault/coordinator/server"
)

func InitCommand() error {
fmt.Println("ArcVault Coordinator - Initialization")
fmt.Println("=====================================\n")

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
fmt.Print("Enter database path (default %USERPROFILE%\.arcvault\arcvault.db): ")
dbPath, _ := reader.ReadString('\n')
dbPath = strings.TrimSpace(dbPath)
if dbPath == "" {
homeDir, _ := os.UserHomeDir()
dbPath = homeDir + "\\.arcvault\\arcvault.db"
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

// Save configuration
if err := config.Save(cfg); err != nil {
return err
}

configPath, _ := config.GetConfigPath()
fmt.Printf("\n✓ Configuration saved to: %s\n", configPath)
fmt.Printf("✓ Database will be initialized at: %s\n", dbPath)
fmt.Printf("✓ Admin token (save this): %s\n\n", token)
fmt.Println("Next step: Run 'coordinator start'")

return nil
}

func StartCommand(cfg *config.Config) error {
log.Printf("Starting ArcVault Coordinator on port %d\n", cfg.Port)

// Initialize database
database, err := db.Init(cfg.DatabasePath)
if err != nil {
return fmt.Errorf("failed to initialize database: %v", err)
}
defer database.Close()

log.Println("✓ Database initialized")

// Start HTTP server
srv := server.New(cfg, database)
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