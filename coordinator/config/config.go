package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Port                             int                 `json:"port"`
	DatabasePath                     string              `json:"database_path"`
	AdminToken                       string              `json:"admin_token"`
	JWTSecret                        string              `json:"jwt_secret"`
	CredentialKey                    string              `json:"credential_key,omitempty"`
	Environment                      string              `json:"environment"`
	CoordinatorID                    string              `json:"coordinator_id,omitempty"`
	AlertHistoryRetentionDays        int                 `json:"alert_history_retention_days,omitempty"`
	Notifications                    *NotificationConfig `json:"notifications,omitempty"`
	Federation                       *FederationConfig   `json:"federation,omitempty"`
	Host                             string              `json:"host,omitempty"`
	CertFile                         string              `json:"cert_file,omitempty"`
	KeyFile                          string              `json:"key_file,omitempty"`
	ExternalTLS                      bool                `json:"external_tls,omitempty"`
	InstallerDir                     string              `json:"installer_dir,omitempty"`
	AllowedOrigins                   []string            `json:"allowed_origins,omitempty"`
	WebSocketOriginValidationEnabled bool                `json:"websocket_origin_validation_enabled"`
}

type NotificationConfig struct {
	OnFailure bool           `json:"on_failure"`
	Webhook   *WebhookConfig `json:"webhook,omitempty"`
	Email     *EmailConfig   `json:"email,omitempty"`
	Slack     *SlackConfig   `json:"slack,omitempty"`
	Teams     *TeamsConfig   `json:"teams,omitempty"`
}

type WebhookConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}

type EmailConfig struct {
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type TeamsConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type FederationConfig struct {
	RootURL string `json:"root_url"`
	Token   string `json:"token"`
}

func GetConfigPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "config.json"), nil
}

func Save(cfg *Config) error {
	// Create sanitized copy for file storage
	// Never write AdminToken or JWTSecret to disk
	sanitized := *cfg
	sanitized.AdminToken = ""
	sanitized.JWTSecret = ""

	path, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("could not determine config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := json.MarshalIndent(&sanitized, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	log.Printf("[config] Sensitive fields (AdminToken, JWTSecret) cleared from config file")
	log.Printf("[config] Set ARCVAULT_ADMIN_TOKEN and ARCVAULT_JWT_SECRET environment variables")

	return os.WriteFile(path, data, 0600)
}

func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config (run 'coordinator init' first): %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}

	// Load sensitive fields from environment variables (override config file)
	if envToken := os.Getenv("ARCVAULT_ADMIN_TOKEN"); envToken != "" {
		cfg.AdminToken = envToken
		log.Printf("[config] AdminToken loaded from ARCVAULT_ADMIN_TOKEN env var")
	}

	if envSecret := os.Getenv("ARCVAULT_JWT_SECRET"); envSecret != "" {
		cfg.JWTSecret = envSecret
		log.Printf("[config] JWTSecret loaded from ARCVAULT_JWT_SECRET env var")
	}

	// Auto-generate JWT secret if still missing after env check
	if cfg.JWTSecret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("could not generate JWT secret: %w", err)
		}
		cfg.JWTSecret = secret
		log.Printf("[config] Generated new JWTSecret (set ARCVAULT_JWT_SECRET to override)")
	}

	// Load WebSocket origin validation kill-switch from environment (default: enabled)
	if envVal := os.Getenv("ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED"); envVal != "" {
		cfg.WebSocketOriginValidationEnabled = envVal != "false"
		log.Printf("[config] WebSocket origin validation: %v (from ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED)", cfg.WebSocketOriginValidationEnabled)
	} else {
		cfg.WebSocketOriginValidationEnabled = true
		log.Printf("[config] WebSocket origin validation: enabled (default)")
	}

	// Validate CORS configuration
	if err := cfg.ValidateAllowedOrigins(); err != nil {
		return nil, fmt.Errorf("CORS configuration invalid: %w", err)
	}

	// Set sensible defaults if AllowedOrigins not configured
	if len(cfg.AllowedOrigins) == 0 {
		if cfg.Environment != "production" {
			// Development: allow localhost by default
			cfg.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
			log.Printf("[config] Development: Using default AllowedOrigins: %v", cfg.AllowedOrigins)
		}
	}

	// Validate production requirements
	if cfg.Environment == "production" && len(cfg.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("AllowedOrigins must be explicitly configured in production (cannot be empty or wildcard)")
	}

	if cfg.Environment == "production" {
		if cfg.AdminToken == "" {
			return nil, fmt.Errorf("CRITICAL: AdminToken not set. Set ARCVAULT_ADMIN_TOKEN environment variable before starting production server")
		}
	}

	return &cfg, nil
}

// ValidateAllowedOrigins checks CORS configuration for security issues.
// Returns error if:
//   - Origins contains wildcard "*" in production mode
//   - Origins contains non-HTTPS URLs (except localhost)
//   - Origins list is empty in production mode
func (c *Config) ValidateAllowedOrigins() error {
	if len(c.AllowedOrigins) == 0 {
		// Development: allow unspecified origins (will use default)
		// Production: enforced in Load() after defaults set
		return nil
	}

	for _, origin := range c.AllowedOrigins {
		// Reject wildcard in all modes
		if origin == "*" {
			return fmt.Errorf("AllowedOrigins cannot contain wildcard '*' — specify explicit domains (e.g., https://dashboard.example.com)")
		}

		// Reject non-https origins except localhost
		if !strings.HasPrefix(origin, "https://") && !strings.HasPrefix(origin, "http://") {
			return fmt.Errorf("AllowedOrigins must use https:// or http://localhost, got: %s", origin)
		}

		if strings.HasPrefix(origin, "http://") {
			if !strings.Contains(origin, "localhost") && !strings.Contains(origin, "127.0.0.1") {
				return fmt.Errorf("non-HTTPS origins only allowed for localhost, got: %s", origin)
			}
		}
	}

	return nil
}

// generateSecret creates a random hex string of the specified byte length.
func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
