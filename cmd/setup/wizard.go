package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// Component represents selected installation components
type Component string

const (
	ComponentCoordinator Component = "coordinator"
	ComponentAgent       Component = "agent"
	ComponentBoth        Component = "both"
)

// SetupConfig holds all configuration values gathered from user
type SetupConfig struct {
	Components Component

	// Coordinator config
	CoordinatorPort int
	AdminUsername   string
	AdminPassword   string
	AdminToken      string
	HTTPS           bool

	// Agent config
	CoordinatorURL string
	AgentID        string
	AgentToken     string
}

// CoordinatorConfig represents the coordinator config.json structure
type CoordinatorConfig struct {
	Port  int    `json:"port"`
	Admin Admin  `json:"admin"`
	Token string `json:"admin_token"`
	HTTPS bool   `json:"https,omitempty"`
}

// Admin holds coordinator admin credentials
type Admin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AgentConfig represents the agent config.yaml structure
type AgentConfig struct {
	CoordinatorURL string `json:"coordinator_url"`
	AgentID        string `json:"agent_id"`
	AuthToken      string `json:"auth_token"`
}

func selectComponents() (Component, error) {
	fmt.Println("Select components to install:")
	fmt.Println("  1) Coordinator (server)")
	fmt.Println("  2) Agent (client)")
	fmt.Println("  3) Both")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter choice (1-3): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	switch input {
	case "1":
		return ComponentCoordinator, nil
	case "2":
		return ComponentAgent, nil
	case "3":
		return ComponentBoth, nil
	default:
		return "", fmt.Errorf("invalid selection: %s", input)
	}
}

func getInstallPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/opt/arcvault"
	}

	switch {
	case isWindows():
		return filepath.Join(homeDir, "ArcVault")
	case isDarwin():
		return "/Applications/ArcVault"
	default:
		return "/opt/arcvault"
	}
}

func gatherConfiguration(components Component) (*SetupConfig, error) {
	config := &SetupConfig{
		Components: components,
	}

	reader := bufio.NewReader(os.Stdin)

	switch components {
	case ComponentCoordinator:
		if err := gatherCoordinatorConfig(reader, config); err != nil {
			return nil, err
		}

	case ComponentAgent:
		if err := gatherAgentConfig(reader, config); err != nil {
			return nil, err
		}

	case ComponentBoth:
		if err := gatherCoordinatorConfig(reader, config); err != nil {
			return nil, err
		}
		fmt.Println()
		fmt.Println("Agent Configuration")
		fmt.Println("==================")
		// Pre-fill agent config with local coordinator details
		config.CoordinatorURL = fmt.Sprintf("http://localhost:%d", config.CoordinatorPort)
		// Auto-generate agent token
		token, err := generateToken(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate agent token: %v", err)
		}
		config.AgentToken = token
		fmt.Printf("Coordinator URL (auto-filled): %s\n", config.CoordinatorURL)
		fmt.Printf("Agent Token (auto-generated): %s\n", config.AgentToken)
		fmt.Print("Enter agent ID (default: hostname): ")
		agentID, _ := reader.ReadString('\n')
		agentID = strings.TrimSpace(agentID)
		if agentID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				agentID = "agent-1"
			} else {
				agentID = hostname
			}
		}
		config.AgentID = agentID
	}

	return config, nil
}

func gatherCoordinatorConfig(reader *bufio.Reader, config *SetupConfig) error {
	fmt.Println()
	fmt.Println("Coordinator Configuration")
	fmt.Println("========================")

	// Port
	fmt.Print("Enter port (default 8080): ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	config.CoordinatorPort = 8080
	if portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %v", err)
		}
		if p < 1 || p > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		config.CoordinatorPort = p
	}

	// Admin Username
	fmt.Print("Enter admin username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("admin username cannot be empty")
	}
	config.AdminUsername = username

	// Admin Password with strength indicator
	fmt.Print("Enter admin password: ")
	password := readPassword()
	if password == "" {
		return fmt.Errorf("admin password cannot be empty")
	}
	strength := evaluatePasswordStrength(password)
	fmt.Printf("Password strength: %s\n", strength)

	config.AdminPassword = password

	// Generate admin token
	token, err := generateToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate admin token: %v", err)
	}
	config.AdminToken = token

	// HTTPS (optional)
	fmt.Print("Enable HTTPS? (y/N): ")
	httpsStr, _ := reader.ReadString('\n')
	httpsStr = strings.TrimSpace(strings.ToLower(httpsStr))
	config.HTTPS = httpsStr == "y" || httpsStr == "yes"

	return nil
}

func gatherAgentConfig(reader *bufio.Reader, config *SetupConfig) error {
	fmt.Println()
	fmt.Println("Agent Configuration")
	fmt.Println("===================")

	// Coordinator URL
	fmt.Print("Enter coordinator URL (e.g., http://coordinator.local:8080): ")
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("coordinator URL cannot be empty")
	}
	config.CoordinatorURL = url

	// Agent ID
	hostname, _ := os.Hostname()
	fmt.Printf("Enter agent ID (default: %s): ", hostname)
	agentID, _ := reader.ReadString('\n')
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		agentID = hostname
	}
	config.AgentID = agentID

	// Auth Token
	fmt.Print("Enter auth token (paste from coordinator): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("auth token cannot be empty")
	}
	config.AgentToken = token

	return nil
}

func reviewSummary(components Component, config *SetupConfig) error {
	fmt.Println()
	fmt.Println("Review Configuration")
	fmt.Println("====================")

	switch components {
	case ComponentCoordinator:
		fmt.Printf("Component: Coordinator (server)\n")
		fmt.Printf("Port: %d\n", config.CoordinatorPort)
		fmt.Printf("Admin Username: %s\n", config.AdminUsername)
		fmt.Printf("HTTPS: %v\n", config.HTTPS)

	case ComponentAgent:
		fmt.Printf("Component: Agent (client)\n")
		fmt.Printf("Coordinator URL: %s\n", config.CoordinatorURL)
		fmt.Printf("Agent ID: %s\n", config.AgentID)

	case ComponentBoth:
		fmt.Printf("Component: Both (Coordinator + Agent)\n")
		fmt.Printf("Coordinator Port: %d\n", config.CoordinatorPort)
		fmt.Printf("Admin Username: %s\n", config.AdminUsername)
		fmt.Printf("HTTPS: %v\n", config.HTTPS)
		fmt.Printf("Agent ID: %s\n", config.AgentID)
		fmt.Printf("Coordinator URL (Agent): %s\n", config.CoordinatorURL)
	}

	fmt.Println()
	fmt.Print("Proceed with installation? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "y" && input != "yes" {
		return fmt.Errorf("user cancelled setup")
	}

	return nil
}

func writeConfigurations(components Component, config *SetupConfig, installPath string) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".arcvault")
	if configDir == filepath.Join("", ".arcvault") {
		// Fallback if HOME is not set
		u, err := user.Current()
		if err != nil {
			configDir = "/opt/arcvault/config"
		} else {
			configDir = filepath.Join(u.HomeDir, ".arcvault")
		}
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Write coordinator config if needed
	if components == ComponentCoordinator || components == ComponentBoth {
		coordConfig := CoordinatorConfig{
			Port: config.CoordinatorPort,
			Admin: Admin{
				Username: config.AdminUsername,
				Password: config.AdminPassword,
			},
			Token: config.AdminToken,
			HTTPS: config.HTTPS,
		}

		configPath := filepath.Join(configDir, "coordinator-config.json")
		data, err := json.MarshalIndent(coordConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal coordinator config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write coordinator config: %v", err)
		}

		fmt.Printf("✓ Coordinator config written to %s\n", configPath)
	}

	// Write agent config if needed
	if components == ComponentAgent || components == ComponentBoth {
		agentConfig := AgentConfig{
			CoordinatorURL: config.CoordinatorURL,
			AgentID:        config.AgentID,
			AuthToken:      config.AgentToken,
		}

		configPath := filepath.Join(configDir, "agent-config.json")
		data, err := json.MarshalIndent(agentConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal agent config: %v", err)
		}

		if err := os.WriteFile(configPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write agent config: %v", err)
		}

		fmt.Printf("✓ Agent config written to %s\n", configPath)
	}

	return nil
}

// Utility functions

func generateToken(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func readPassword() string {
	// Simple password read (not hidden on all platforms)
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	return strings.TrimSpace(password)
}

func evaluatePasswordStrength(password string) string {
	score := 0

	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, ch := range password {
		if unicode.IsLower(ch) {
			hasLower = true
		}
		if unicode.IsUpper(ch) {
			hasUpper = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
		if unicode.IsPunct(ch) || unicode.IsSymbol(ch) {
			hasSpecial = true
		}
	}

	if hasLower {
		score++
	}
	if hasUpper {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	switch {
	case score <= 2:
		return "Weak"
	case score <= 4:
		return "Fair"
	case score <= 6:
		return "Good"
	default:
		return "Strong"
	}
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}

func isDarwin() bool {
	// Check if running on macOS
	return os.Getenv("OSTYPE") == "darwin" || strings.HasPrefix(os.Getenv("OSTYPE"), "darwin")
}
