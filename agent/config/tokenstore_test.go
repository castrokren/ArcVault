package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeCfg(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent-config.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// The exchanged token has to survive a restart. If it is only held in memory the
// agent falls back to the enrollment token in the file, which expires an hour
// after the install script was generated.
func TestTokenStore_ReplacePersistsAndPreservesOtherKeys(t *testing.T) {
	path := writeCfg(t, strings.Join([]string{
		"agent_id: HOST-A",
		"coordinator_url: https://coordinator.example",
		"auth_token: enrollment-token",
		"ca_cert_file: C:\\ArcVault-Agent\\coordinator.crt",
		"",
	}, "\n"))

	s := NewTokenStore("enrollment-token", path)
	if err := s.Replace("per-agent-token"); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if got := s.Get(); got != "per-agent-token" {
		t.Errorf("in-memory token = %q, want per-agent-token", got)
	}

	// Reload through the real parser: proves the file is still valid YAML.
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("config no longer loads after Replace: %v", err)
	}
	if cfg.AuthToken != "per-agent-token" {
		t.Errorf("persisted auth_token = %q, want per-agent-token", cfg.AuthToken)
	}
	if cfg.AgentID != "HOST-A" {
		t.Errorf("agent_id was lost: %q", cfg.AgentID)
	}
	if cfg.CoordinatorURL != "https://coordinator.example" {
		t.Errorf("coordinator_url was lost: %q", cfg.CoordinatorURL)
	}
	if cfg.CACertFile != "C:\\ArcVault-Agent\\coordinator.crt" {
		t.Errorf("ca_cert_file was lost: %q — the agent would stop trusting the coordinator", cfg.CACertFile)
	}
}

// A missing auth_token line must be appended, not silently dropped.
func TestTokenStore_ReplaceAppendsWhenKeyAbsent(t *testing.T) {
	path := writeCfg(t, "agent_id: HOST-A\ncoordinator_url: https://c.example\n")

	s := NewTokenStore("", path)
	if err := s.Replace("per-agent-token"); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthToken != "per-agent-token" {
		t.Errorf("auth_token = %q, want per-agent-token", cfg.AuthToken)
	}
}

// An empty token must never overwrite a working one — Load() rejects an empty
// auth_token, so persisting one would make the agent fail to start.
func TestTokenStore_ReplaceRejectsEmpty(t *testing.T) {
	path := writeCfg(t, "agent_id: HOST-A\ncoordinator_url: https://c.example\nauth_token: good-token\n")

	s := NewTokenStore("good-token", path)
	if err := s.Replace(""); err == nil {
		t.Error("Replace(\"\") should error")
	}
	if got := s.Get(); got != "good-token" {
		t.Errorf("token was clobbered: %q", got)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("config broken by empty Replace: %v", err)
	}
	if cfg.AuthToken != "good-token" {
		t.Errorf("persisted token = %q, want good-token", cfg.AuthToken)
	}
}

// Only the auth_token line may change. A regex anchored loosely could match
// something else in the file.
func TestTokenStore_ReplaceRewritesOnlyTheTokenLine(t *testing.T) {
	path := writeCfg(t, strings.Join([]string{
		"# ArcVault agent config",
		"agent_id: HOST-A",
		"coordinator_url: https://c.example",
		"auth_token: old",
		"honcho_url: http://localhost:8000",
		"",
	}, "\n"))

	s := NewTokenStore("old", path)
	if err := s.Replace("new"); err != nil {
		t.Fatalf("Replace: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"# ArcVault agent config",
		"agent_id: HOST-A",
		"auth_token: new",
		"honcho_url: http://localhost:8000",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q to survive, file is:\n%s", want, got)
		}
	}
	if strings.Contains(got, "auth_token: old") {
		t.Error("old token still present in the file")
	}
}

// The heartbeat loop, job runner and WS client all read concurrently while
// registration writes. Run with -race for this to mean anything.
func TestTokenStore_ConcurrentGetAndReplace(t *testing.T) {
	path := writeCfg(t, "agent_id: A\ncoordinator_url: https://c\nauth_token: t0\n")
	s := NewTokenStore("t0", path)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Get()
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Replace("t1")
	}()
	wg.Wait()

	if got := s.Get(); got != "t1" {
		t.Errorf("token = %q, want t1", got)
	}
}
