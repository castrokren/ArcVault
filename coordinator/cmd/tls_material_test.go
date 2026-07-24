package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"arcvault/coordinator/config"
)

// A fresh install writes a config.json with no cert_file/key_file. Before this
// guard, Server.Start() saw empty paths and silently served plain HTTP on 443
// while the installer, dashboard and agents all addressed it as https://.
func TestEnsureTLSMaterial_generatesWhenUnconfigured(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		CertFile: filepath.Join(dir, "cert.pem"),
		KeyFile:  filepath.Join(dir, "key.pem"),
	}

	if err := ensureTLSMaterial(cfg); err != nil {
		t.Fatalf("ensureTLSMaterial: %v", err)
	}
	for _, p := range []string{cfg.CertFile, cfg.KeyFile} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

// Blank host must still yield a usable cert — the installer never sets one.
func TestEnsureTLSMaterial_blankHostStillGenerates(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Host:     "",
		CertFile: filepath.Join(dir, "cert.pem"),
		KeyFile:  filepath.Join(dir, "key.pem"),
	}

	if err := ensureTLSMaterial(cfg); err != nil {
		t.Fatalf("ensureTLSMaterial with blank host: %v", err)
	}
	if _, err := os.Stat(cfg.CertFile); err != nil {
		t.Errorf("expected cert for blank host: %v", err)
	}
}

// Restarts must not churn the cert — a regenerated cert would break every agent
// that pinned the old one.
func TestEnsureTLSMaterial_idempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		CertFile: filepath.Join(dir, "cert.pem"),
		KeyFile:  filepath.Join(dir, "key.pem"),
	}

	if err := ensureTLSMaterial(cfg); err != nil {
		t.Fatalf("first call: %v", err)
	}
	first, err := os.ReadFile(cfg.CertFile)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}

	if err := ensureTLSMaterial(cfg); err != nil {
		t.Fatalf("second call: %v", err)
	}
	second, err := os.ReadFile(cfg.CertFile)
	if err != nil {
		t.Fatalf("re-read cert: %v", err)
	}

	if string(first) != string(second) {
		t.Error("cert was regenerated on the second call; agents pinning it would break")
	}
}

// Upstream terminator owns TLS; the coordinator must not mint a stray cert.
func TestEnsureTLSMaterial_noopWhenExternalTLS(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ExternalTLS: true,
		CertFile:    filepath.Join(dir, "cert.pem"),
		KeyFile:     filepath.Join(dir, "key.pem"),
	}

	if err := ensureTLSMaterial(cfg); err != nil {
		t.Fatalf("ensureTLSMaterial: %v", err)
	}
	if _, err := os.Stat(cfg.CertFile); err == nil {
		t.Error("generated a certificate despite external TLS termination")
	}
}
