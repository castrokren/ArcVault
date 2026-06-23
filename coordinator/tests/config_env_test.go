package tests

import (
	"encoding/json"
	"os"
	"testing"

	"arcvault/coordinator/config"
)

// TestLoad_ReadsEnvironmentVariables tests that environment variables override
// config file values for sensitive fields.
func TestLoad_ReadsEnvironmentVariables(t *testing.T) {
	// Get the config file path that Load() expects
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	// Backup existing config if it exists
	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	// Create a test config file
	testConfig := config.Config{
		Port:           8080,
		DatabasePath:   "/tmp/db",
		AdminToken:     "file-token",
		JWTSecret:      "file-secret",
		Environment:    "development",
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	data, err := json.MarshalIndent(&testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Set environment variables
	os.Setenv("ARCVAULT_ADMIN_TOKEN", "env-token")
	os.Setenv("ARCVAULT_JWT_SECRET", "env-secret")
	defer func() {
		os.Unsetenv("ARCVAULT_ADMIN_TOKEN")
		os.Unsetenv("ARCVAULT_JWT_SECRET")
	}()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify environment variables override file values
	if cfg.AdminToken != "env-token" {
		t.Errorf("AdminToken = %q, want %q", cfg.AdminToken, "env-token")
	}

	if cfg.JWTSecret != "env-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "env-secret")
	}

	// Verify other fields are loaded from file
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}

	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want %q", cfg.Environment, "development")
	}
}

// TestLoad_FailsIfMissingAdminToken tests that production mode requires AdminToken
// from environment variable and fails if missing.
func TestLoad_FailsIfMissingAdminToken(t *testing.T) {
	// Get the config file path that Load() expects
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	// Backup existing config if it exists
	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	// Create a test config file without tokens
	testConfig := config.Config{
		Port:           8080,
		DatabasePath:   "/tmp/db",
		Environment:    "production",
		AllowedOrigins: []string{"https://example.com"},
	}

	data, err := json.MarshalIndent(&testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Ensure environment variables are not set
	os.Unsetenv("ARCVAULT_ADMIN_TOKEN")
	os.Unsetenv("ARCVAULT_JWT_SECRET")

	// Load config should fail in production without AdminToken
	cfg, err := config.Load()
	if err == nil {
		t.Fatalf("Load() should have failed in production without AdminToken, but got config: %v", cfg)
	}

	if cfg != nil {
		t.Errorf("Config should be nil when load fails, got: %+v", cfg)
	}
}

// TestSave_NeverWritesTokensToDisk tests that Save() clears sensitive tokens
// and never persists AdminToken or JWTSecret to the config file.
func TestSave_NeverWritesTokensToDisk(t *testing.T) {
	// Get the config file path that Save() expects
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	// Backup existing config if it exists
	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	// Create a test config with sensitive fields
	testConfig := &config.Config{
		Port:           8080,
		DatabasePath:   "/tmp/db",
		AdminToken:     "super-secret-admin-token",
		JWTSecret:      "super-secret-jwt-secret",
		Environment:    "production",
		AllowedOrigins: []string{"https://example.com"},
	}

	// Save the config
	err = config.Save(testConfig)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Read the saved file directly
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config file: %v", err)
	}

	// Unmarshal the saved config
	var savedConfig config.Config
	if err = json.Unmarshal(data, &savedConfig); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	// Verify sensitive fields are cleared in saved file
	if savedConfig.AdminToken != "" {
		t.Errorf("AdminToken in saved file = %q, want empty string", savedConfig.AdminToken)
	}

	if savedConfig.JWTSecret != "" {
		t.Errorf("JWTSecret in saved file = %q, want empty string", savedConfig.JWTSecret)
	}

	// Verify other fields are preserved
	if savedConfig.Port != 8080 {
		t.Errorf("Port in saved file = %d, want 8080", savedConfig.Port)
	}

	if savedConfig.Environment != "production" {
		t.Errorf("Environment in saved file = %q, want %q", savedConfig.Environment, "production")
	}

	// Check that the saved config file doesn't contain the actual secrets as strings
	configFileContent := string(data)
	if configFileContent != "" {
		if !contains(configFileContent, "super-secret-admin-token") &&
			!contains(configFileContent, "super-secret-jwt-secret") {
			// Good: tokens are not in the file
		} else {
			t.Errorf("Sensitive tokens found in saved config file")
		}
	}
}

// TestLoad_WebSocketOriginValidationKillSwitch tests that the kill-switch
// env var ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED controls origin validation.
func TestLoad_WebSocketOriginValidationKillSwitch(t *testing.T) {
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	testConfig := config.Config{
		Port:        8080,
		DatabasePath: "/tmp/db",
		AdminToken: "test-token",
		Environment: "production",
		AllowedOrigins: []string{"https://example.com"},
	}

	data, err := json.MarshalIndent(&testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test 1: Default (enabled)
	os.Unsetenv("ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if !cfg.WebSocketOriginValidationEnabled {
		t.Errorf("WebSocketOriginValidationEnabled = false, want true (default)")
	}

	// Test 2: Explicitly enabled
	os.Setenv("ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED", "true")
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if !cfg.WebSocketOriginValidationEnabled {
		t.Errorf("WebSocketOriginValidationEnabled = false, want true (explicit)")
	}

	// Test 3: Disabled (kill-switch)
	os.Setenv("ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED", "false")
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.WebSocketOriginValidationEnabled {
		t.Errorf("WebSocketOriginValidationEnabled = true, want false (disabled)")
	}

	defer os.Unsetenv("ARCVAULT_WEBSOCKET_ORIGIN_VALIDATION_ENABLED")
}

// TestLoad_AllowedOrigins_DevMode tests that development mode sets default origins.
func TestLoad_AllowedOrigins_DevMode(t *testing.T) {
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	testConfig := config.Config{
		Port:        8080,
		DatabasePath: "/tmp/db",
		AdminToken: "test-token",
		Environment: "development",
		AllowedOrigins: []string{}, // Empty origins in dev mode
	}

	data, err := json.MarshalIndent(&testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Dev mode should set default localhost origins
	if len(cfg.AllowedOrigins) != 2 {
		t.Errorf("AllowedOrigins count = %d, want 2", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[0] != "http://localhost:5173" {
		t.Errorf("AllowedOrigins[0] = %q, want http://localhost:5173", cfg.AllowedOrigins[0])
	}
	if cfg.AllowedOrigins[1] != "http://localhost:3000" {
		t.Errorf("AllowedOrigins[1] = %q, want http://localhost:3000", cfg.AllowedOrigins[1])
	}
}

// TestLoad_AllowedOrigins_ProductionEmptyFails tests that production mode fails
// if AllowedOrigins is empty (no defaults allowed in production).
func TestLoad_AllowedOrigins_ProductionEmptyFails(t *testing.T) {
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Skipf("Could not get config path: %v", err)
	}

	backupPath := configPath + ".test.backup"
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			t.Fatalf("Failed to backup existing config: %v", err)
		}
		defer func() {
			os.Remove(configPath)
			os.Rename(backupPath, configPath)
		}()
	} else {
		defer os.Remove(configPath)
	}

	testConfig := config.Config{
		Port:        8080,
		DatabasePath: "/tmp/db",
		AdminToken: "test-token",
		Environment: "production",
		AllowedOrigins: []string{}, // Empty origins in prod mode
	}

	data, err := json.MarshalIndent(&testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load should fail in production with empty origins
	cfg, err := config.Load()
	if err == nil {
		t.Errorf("Load() should fail in production with empty AllowedOrigins, got config: %+v", cfg)
	}
}

// Helper function to check if a substring exists in a string
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
