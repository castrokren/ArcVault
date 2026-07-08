package server

import (
	"net/http"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

// TestServerOption is a functional option for configuring test servers.
type TestServerOption func(*testServerConfig)

// testServerConfig holds configuration for building a test server.
type testServerConfig struct {
	cfg          *config.Config
	db           *db.DB
	handlerOnly  bool
	customConfig *config.Config
}

// WithHandlerOnly returns an option that creates a minimal server with no real DB.
// Handlers under test must not touch s.db (it will be empty).
func WithHandlerOnly() TestServerOption {
	return func(c *testServerConfig) {
		c.handlerOnly = true
	}
}

// WithConfig returns an option that uses a custom config.
func WithConfig(cfg *config.Config) TestServerOption {
	return func(c *testServerConfig) {
		c.customConfig = cfg
	}
}

// newTestServer returns a Server configured for testing.
// Default behavior: full integration path with real in-memory SQLite DB.
// Use WithHandlerOnly() + WithConfig() for handler-only tests.
func newTestServer(t *testing.T, opts ...TestServerOption) *Server {
	t.Helper()

	// Apply options to config
	cfg := &testServerConfig{
		cfg:         nil,
		db:          nil,
		handlerOnly: false,
		customConfig: &config.Config{
			Port:       8080,
			AdminToken: "test-token",
			JWTSecret:  "test-secret",
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Use custom config if provided, otherwise use default
	finalCfg := cfg.customConfig
	if cfg.customConfig == nil {
		finalCfg = &config.Config{
			Port:       8080,
			AdminToken: "test-token",
			JWTSecret:  "test-secret",
		}
	}

	// Handle handler-only vs integration paths
	if cfg.handlerOnly {
		// Handler-only: minimal server, no real DB
		return &Server{
			cfg:           finalCfg,
			db:            &db.DB{},
			router:        http.NewServeMux(),
			tokenCache:    make(map[string]tokenCacheEntry),
			loginLimiters: make(map[string]*loginRateLimiter),
		}
	}

	// Integration path: real in-memory SQLite DB
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// Pass empty staticDir so tests don't try to serve files from disk
	return NewWithStatic(finalCfg, database, "")
}
