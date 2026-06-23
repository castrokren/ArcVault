package tests

import (
	"net/http"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/server"
)

// TestInitWebSocketUpgrader_AllowsWhitelistedOrigins tests that the WebSocket
// upgrader allows origins that are in the AllowedOrigins whitelist.
func TestInitWebSocketUpgrader_AllowsWhitelistedOrigins(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		testOrigin     string
		wantAllow      bool
	}{
		{
			name:           "exact match allowed",
			allowedOrigins: []string{"https://dashboard.example.com"},
			testOrigin:     "https://dashboard.example.com",
			wantAllow:      true,
		},
		{
			name:           "multiple origins, one matches",
			allowedOrigins: []string{"https://dashboard.example.com", "https://api.example.com"},
			testOrigin:     "https://api.example.com",
			wantAllow:      true,
		},
		{
			name:           "localhost allowed",
			allowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
			testOrigin:     "http://localhost:3000",
			wantAllow:      true,
		},
		{
			name:           "empty origin allowed",
			allowedOrigins: []string{"https://example.com"},
			testOrigin:     "",
			wantAllow:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:                             8080,
				AllowedOrigins:                   tt.allowedOrigins,
				Environment:                      "development",
				WebSocketOriginValidationEnabled: true,
			}

			// Create a minimal server
			s := &server.Server{}
			if err := initTestServer(s, cfg); err != nil {
				t.Fatalf("Failed to init test server: %v", err)
			}

			// Create a test request with the origin header
			req, _ := http.NewRequest("GET", "http://localhost:8080/ws", nil)
			if tt.testOrigin != "" {
				req.Header.Set("Origin", tt.testOrigin)
			}

			// Test the CheckOrigin function
			allowed := s.GetWebSocketUpgraderInternal().CheckOrigin(req)

			if allowed != tt.wantAllow {
				t.Errorf("CheckOrigin(%q) = %v, want %v", tt.testOrigin, allowed, tt.wantAllow)
			}
		})
	}
}

// TestInitWebSocketUpgrader_RejectsUnlistedOrigins tests that the WebSocket
// upgrader rejects origins that are not in the AllowedOrigins whitelist.
func TestInitWebSocketUpgrader_RejectsUnlistedOrigins(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		testOrigin     string
		wantAllow      bool
		desc           string
	}{
		{
			name:           "unlisted origin rejected",
			allowedOrigins: []string{"https://dashboard.example.com"},
			testOrigin:     "https://attacker.com",
			wantAllow:      false,
			desc:           "origin not in whitelist",
		},
		{
			name:           "wrong port rejected",
			allowedOrigins: []string{"http://localhost:3000"},
			testOrigin:     "http://localhost:5173",
			wantAllow:      false,
			desc:           "port mismatch is considered different origin",
		},
		{
			name:           "http vs https rejected",
			allowedOrigins: []string{"https://example.com"},
			testOrigin:     "http://example.com",
			wantAllow:      false,
			desc:           "scheme mismatch is considered different origin",
		},
		{
			name:           "subdomain mismatch rejected",
			allowedOrigins: []string{"https://dashboard.example.com"},
			testOrigin:     "https://api.example.com",
			wantAllow:      false,
			desc:           "different subdomain is different origin",
		},
		{
			name:           "partial match rejected",
			allowedOrigins: []string{"https://example.com"},
			testOrigin:     "https://example.com.attacker.com",
			wantAllow:      false,
			desc:           "substring matches are not allowed",
		},
		{
			name:           "case sensitive match",
			allowedOrigins: []string{"https://Example.Com"},
			testOrigin:     "https://example.com",
			wantAllow:      false,
			desc:           "origin matching is case sensitive (exact string match)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:                             8080,
				AllowedOrigins:                   tt.allowedOrigins,
				Environment:                      "development",
				WebSocketOriginValidationEnabled: true,
			}

			// Create a minimal server
			s := &server.Server{}
			if err := initTestServer(s, cfg); err != nil {
				t.Fatalf("Failed to init test server: %v", err)
			}

			// Create a test request with the origin header
			req, _ := http.NewRequest("GET", "http://localhost:8080/ws", nil)
			if tt.testOrigin != "" {
				req.Header.Set("Origin", tt.testOrigin)
			}

			// Test the CheckOrigin function
			allowed := s.GetWebSocketUpgraderInternal().CheckOrigin(req)

			if allowed != tt.wantAllow {
				t.Errorf("CheckOrigin(%q) = %v, want %v (case: %s)", tt.testOrigin, allowed, tt.wantAllow, tt.desc)
			}
		})
	}
}

// TestInitWebSocketUpgrader_FailSafeDefault tests that the WebSocket upgrader
// has sensible defaults when AllowedOrigins is empty or nil.
func TestInitWebSocketUpgrader_FailSafeDefault(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		testOrigin     string
		wantAllow      bool
		desc           string
	}{
		{
			name:           "empty origins, no origin header",
			allowedOrigins: []string{},
			testOrigin:     "",
			wantAllow:      true,
			desc:           "no origin header always allowed (same-origin request)",
		},
		{
			name:           "empty origins, origin header present",
			allowedOrigins: []string{},
			testOrigin:     "https://example.com",
			wantAllow:      false,
			desc:           "origin header with empty whitelist should be rejected",
		},
		{
			name:           "nil origins, no origin header",
			allowedOrigins: nil,
			testOrigin:     "",
			wantAllow:      true,
			desc:           "nil list treated like empty, no header allowed",
		},
		{
			name:           "nil origins, origin header present",
			allowedOrigins: nil,
			testOrigin:     "https://example.com",
			wantAllow:      false,
			desc:           "nil list treated like empty, origin header rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:                             8080,
				AllowedOrigins:                   tt.allowedOrigins,
				Environment:                      "development",
				WebSocketOriginValidationEnabled: true,
			}

			// Create a minimal server
			s := &server.Server{}
			if err := initTestServer(s, cfg); err != nil {
				t.Fatalf("Failed to init test server: %v", err)
			}

			// Create a test request with the origin header
			req, _ := http.NewRequest("GET", "http://localhost:8080/ws", nil)
			if tt.testOrigin != "" {
				req.Header.Set("Origin", tt.testOrigin)
			}

			// Test the CheckOrigin function
			allowed := s.GetWebSocketUpgraderInternal().CheckOrigin(req)

			if allowed != tt.wantAllow {
				t.Errorf("CheckOrigin(%q) = %v, want %v (case: %s)", tt.testOrigin, allowed, tt.wantAllow, tt.desc)
			}
		})
	}
}

// TestInitWebSocketUpgrader_KillSwitchBypassesValidation tests that setting
// WebSocketOriginValidationEnabled to false bypasses all origin checks (for emergency rollback).
func TestInitWebSocketUpgrader_KillSwitchBypassesValidation(t *testing.T) {
	tests := []struct {
		name                    string
		killSwitchEnabled       bool
		allowedOrigins          []string
		testOrigin              string
		wantAllow               bool
		desc                    string
	}{
		{
			name:              "kill-switch disabled, allowed origin",
			killSwitchEnabled: true,
			allowedOrigins:    []string{"https://dashboard.example.com"},
			testOrigin:        "https://dashboard.example.com",
			wantAllow:         true,
			desc:              "normal validation works",
		},
		{
			name:              "kill-switch disabled, unlisted origin",
			killSwitchEnabled: true,
			allowedOrigins:    []string{"https://dashboard.example.com"},
			testOrigin:        "https://attacker.com",
			wantAllow:         false,
			desc:              "unlisted origin rejected when validation enabled",
		},
		{
			name:              "kill-switch enabled, unlisted origin",
			killSwitchEnabled: false,
			allowedOrigins:    []string{"https://dashboard.example.com"},
			testOrigin:        "https://attacker.com",
			wantAllow:         true,
			desc:              "kill-switch allows any origin (emergency rollback)",
		},
		{
			name:              "kill-switch enabled, empty origins",
			killSwitchEnabled: false,
			allowedOrigins:    []string{},
			testOrigin:        "https://any.origin.com",
			wantAllow:         true,
			desc:              "kill-switch allows any origin even with empty list",
		},
		{
			name:              "kill-switch enabled, no origin header",
			killSwitchEnabled: false,
			allowedOrigins:    []string{"https://example.com"},
			testOrigin:        "",
			wantAllow:         true,
			desc:              "no origin header always allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:                             8080,
				AllowedOrigins:                   tt.allowedOrigins,
				Environment:                      "production",
				WebSocketOriginValidationEnabled: tt.killSwitchEnabled,
			}

			s := &server.Server{}
			if err := initTestServer(s, cfg); err != nil {
				t.Fatalf("Failed to init test server: %v", err)
			}

			req, _ := http.NewRequest("GET", "http://localhost:8080/ws", nil)
			if tt.testOrigin != "" {
				req.Header.Set("Origin", tt.testOrigin)
			}

			allowed := s.GetWebSocketUpgraderInternal().CheckOrigin(req)

			if allowed != tt.wantAllow {
				t.Errorf("CheckOrigin(%q) with kill-switch=%v = %v, want %v (case: %s)",
					tt.testOrigin, !tt.killSwitchEnabled, allowed, tt.wantAllow, tt.desc)
			}
		})
	}
}

// ============ Helper Functions ============

// initTestServer initializes a minimal Server for testing purposes.
// Only sets up the wsUpgrader field which is used by CheckOrigin tests.
func initTestServer(s *server.Server, cfg *config.Config) error {
	s.SetConfigInternal(cfg)
	s.InitWebSocketUpgraderInternal()
	return nil
}

