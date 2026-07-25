package config

import (
	"fmt"
	"os"
	"regexp"
	"sync"
)

// authTokenLine matches the auth_token entry in agent-config.yaml.
var authTokenLine = regexp.MustCompile(`(?m)^[ \t]*auth_token:.*$`)

// TokenStore holds the agent's current auth token.
//
// It exists because registration can *exchange* credentials: a machine enrolled
// via bootstrap.ps1 starts with a short-lived enrollment token and receives a
// long-lived per-agent token back from POST /api/agents/register. Every consumer
// therefore has to read the token at request time. The heartbeat loop, the job
// runner and the WebSocket client used to copy the string at construction, which
// left them holding the enrollment token after it had been replaced — and that
// token expires an hour after the install script was generated.
type TokenStore struct {
	mu         sync.RWMutex
	token      string
	configPath string
}

// NewTokenStore returns a store seeded with the token from configPath.
func NewTokenStore(token, configPath string) *TokenStore {
	return &TokenStore{token: token, configPath: configPath}
}

// Get returns the current token. Safe for concurrent use.
func (s *TokenStore) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

// Replace swaps in a new token and persists it to agent-config.yaml.
//
// The in-memory swap happens even when the write fails, so the current process
// keeps working with the credential the coordinator just issued; the error is
// returned for the caller to log. A failed write means the next restart falls
// back to the old token in the file, which is why the coordinator lets the
// enrollment token expire on its own rather than deleting it on exchange.
//
// Rewrites only the auth_token line so comments and any other keys survive.
func (s *TokenStore) Replace(token string) error {
	if token == "" {
		return fmt.Errorf("refusing to store an empty auth token")
	}

	s.mu.Lock()
	unchanged := s.token == token
	s.token = token
	path := s.configPath
	s.mu.Unlock()

	if unchanged || path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read %s to persist new token: %w", path, err)
	}

	// ReplaceAllLiteral: a token must never be treated as a $-expansion.
	line := []byte("auth_token: " + token)
	if authTokenLine.Match(data) {
		data = authTokenLine.ReplaceAllLiteral(data, line)
	} else {
		data = append(data, append([]byte("\n"), append(line, '\n')...)...)
	}

	// 0600: the file holds a credential.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}
