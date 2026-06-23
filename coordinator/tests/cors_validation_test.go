package tests

import (
	"testing"

	"arcvault/coordinator/config"
)

// TestValidateAllowedOrigins_RejectsWildcard tests that wildcard origins are rejected
// in all modes (production and development).
func TestValidateAllowedOrigins_RejectsWildcard(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		origins     []string
		wantErr     bool
	}{
		{
			name:        "wildcard in production rejected",
			environment: "production",
			origins:     []string{"*"},
			wantErr:     true,
		},
		{
			name:        "wildcard in development rejected",
			environment: "development",
			origins:     []string{"*"},
			wantErr:     true,
		},
		{
			name:        "wildcard with other origins rejected",
			environment: "development",
			origins:     []string{"https://example.com", "*"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Environment:   tt.environment,
				AllowedOrigins: tt.origins,
			}

			err := cfg.ValidateAllowedOrigins()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllowedOrigins() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateAllowedOrigins_EnforcesHTTPS tests that non-HTTPS origins are only
// allowed for localhost/127.0.0.1.
func TestValidateAllowedOrigins_EnforcesHTTPS(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		origins     []string
		wantErr     bool
		desc        string
	}{
		{
			name:        "http://localhost allowed",
			environment: "development",
			origins:     []string{"http://localhost:3000"},
			wantErr:     false,
			desc:        "localhost over HTTP is allowed",
		},
		{
			name:        "http://127.0.0.1 allowed",
			environment: "development",
			origins:     []string{"http://127.0.0.1:3000"},
			wantErr:     false,
			desc:        "127.0.0.1 over HTTP is allowed",
		},
		{
			name:        "http://example.com rejected",
			environment: "production",
			origins:     []string{"http://example.com"},
			wantErr:     true,
			desc:        "non-localhost domains must use HTTPS",
		},
		{
			name:        "https://example.com allowed",
			environment: "production",
			origins:     []string{"https://example.com"},
			wantErr:     false,
			desc:        "HTTPS domains are always allowed",
		},
		{
			name:        "mixed http and https",
			environment: "development",
			origins:     []string{"http://localhost:3000", "https://api.example.com"},
			wantErr:     false,
			desc:        "localhost HTTP and HTTPS domains together",
		},
		{
			name:        "http domain without localhost rejected",
			environment: "production",
			origins:     []string{"http://api.example.com", "https://example.com"},
			wantErr:     true,
			desc:        "first origin rejected, error returned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Environment:   tt.environment,
				AllowedOrigins: tt.origins,
			}

			err := cfg.ValidateAllowedOrigins()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllowedOrigins() error = %v, wantErr %v (case: %s)", err, tt.wantErr, tt.desc)
			}
		})
	}
}

// TestValidateAllowedOrigins_AllowsLocalhost tests that localhost variations are
// properly allowed.
func TestValidateAllowedOrigins_AllowsLocalhost(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		wantErr bool
	}{
		{
			name:    "localhost with default ports",
			origins: []string{"http://localhost:3000", "http://localhost:5173"},
			wantErr: false,
		},
		{
			name:    "127.0.0.1 variations",
			origins: []string{"http://127.0.0.1:3000", "http://127.0.0.1:5173"},
			wantErr: false,
		},
		{
			name:    "mixed localhost forms",
			origins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
			wantErr: false,
		},
		{
			name:    "localhost with https",
			origins: []string{"https://localhost:3000"},
			wantErr: false,
		},
		{
			name:    "single localhost",
			origins: []string{"http://localhost"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Environment:   "development",
				AllowedOrigins: tt.origins,
			}

			err := cfg.ValidateAllowedOrigins()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllowedOrigins() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateAllowedOrigins_DevelopmentDefaults tests development mode defaults
// and empty origin list handling.
func TestValidateAllowedOrigins_DevelopmentDefaults(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		origins     []string
		wantErr     bool
		desc        string
	}{
		{
			name:        "empty origins in development",
			environment: "development",
			origins:     []string{},
			wantErr:     false,
			desc:        "empty list allowed in development",
		},
		{
			name:        "empty origins in production",
			environment: "production",
			origins:     []string{},
			wantErr:     false,
			desc:        "validation allows empty (Load() enforces it separately)",
		},
		{
			name:        "nil origins in development",
			environment: "development",
			origins:     nil,
			wantErr:     false,
			desc:        "nil treated as empty",
		},
		{
			name:        "valid production origins",
			environment: "production",
			origins:     []string{"https://dashboard.example.com", "https://api.example.com"},
			wantErr:     false,
			desc:        "multiple HTTPS origins allowed in production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Environment:   tt.environment,
				AllowedOrigins: tt.origins,
			}

			err := cfg.ValidateAllowedOrigins()

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllowedOrigins() error = %v, wantErr %v (case: %s)", err, tt.wantErr, tt.desc)
			}
		})
	}
}
