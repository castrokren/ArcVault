package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port                       int                 `json:"port"`
	DatabasePath               string              `json:"database_path"`
	AdminToken                 string              `json:"admin_token"`
	JWTSecret                  string              `json:"jwt_secret"`
	CredentialKey              string              `json:"credential_key,omitempty"`
	Environment                string              `json:"environment"`
	CoordinatorID              string              `json:"coordinator_id,omitempty"`
	AlertHistoryRetentionDays  int                 `json:"alert_history_retention_days,omitempty"`
	Notifications              *NotificationConfig `json:"notifications,omitempty"`
	Federation                 *FederationConfig   `json:"federation,omitempty"`
	Host                       string              `json:"host,omitempty"`
	CertFile                   string              `json:"cert_file,omitempty"`
	KeyFile                    string              `json:"key_file,omitempty"`
	ExternalTLS                bool                `json:"external_tls,omitempty"`
	InstallerDir               string              `json:"installer_dir,omitempty"`
	AllowedOrigins             []string            `json:"allowed_origins,omitempty"`
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
	path, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("could not determine config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
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

	// Auto-generate JWT secret if missing
	if cfg.JWTSecret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return nil, fmt.Errorf("could not generate JWT secret: %w", err)
		}
		cfg.JWTSecret = secret
		// Save updated config back to file
		if err := Save(&cfg); err != nil {
			return nil, fmt.Errorf("could not save updated config: %w", err)
		}
	}

	return &cfg, nil
}

// generateSecret creates a random hex string of the specified byte length.
func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
