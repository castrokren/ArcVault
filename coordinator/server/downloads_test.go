package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

// newTestServer returns a minimal Server wired for handler tests.
// No real DB connection — handlers under test must not touch s.db.
func newTestServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	return &Server{
		cfg:           cfg,
		db:            &db.DB{},
		router:        http.NewServeMux(),
		tokenCache:    make(map[string]tokenCacheEntry),
		loginLimiters: make(map[string]*loginRateLimiter),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// REGRESSION: /downloads/installer must serve the .exe, not bootstrap.ps1
//
// This test exists because the route was accidentally wired to handleBootstrapScript
// in the past. If this test fails, re-check registerRoutes() in server.go.
// ─────────────────────────────────────────────────────────────────────────────

// TestHandleDownloadInstaller_ServesExe verifies the handler sends back a binary
// .exe, not a PowerShell script. Tests the handler directly (auth is separate).
func TestHandleDownloadInstaller_ServesExe(t *testing.T) {
	// Build a temp dir containing a fake installer exe.
	tmpDir := t.TempDir()
	version := strings.TrimPrefix(Version, "v")
	if version == "" || version == "dev" {
		version = "0.5.1"
	}
	exeName := fmt.Sprintf("ArcVault-Setup-%s-windows-amd64.exe", version)
	if err := os.WriteFile(filepath.Join(tmpDir, exeName), []byte("FAKE_EXE"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Setenv("ARCVAULT_VERSION", "v"+version)

	srv := newTestServer(t, &config.Config{
		Port:         8080,
		InstallerDir: tmpDir,
	})

	w := httptest.NewRecorder()
	srv.handleDownloadInstaller(w, httptest.NewRequest(http.MethodGet, "/downloads/installer", nil))
	resp := w.Result()

	// ── Must succeed ──────────────────────────────────────────────────────────
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("[REGRESSION] handleDownloadInstaller returned %d, want 200. Body: %s",
			resp.StatusCode, w.Body.String())
	}

	// ── Content-Type must be binary, not text/plain ───────────────────────────
	ct := resp.Header.Get("Content-Type")
	if ct != "application/octet-stream" {
		t.Errorf("[REGRESSION] Content-Type = %q, want application/octet-stream.\n"+
			"If 'text/plain', the route was accidentally wired to handleBootstrapScript.", ct)
	}

	// ── filename must end in .exe, not .ps1 ──────────────────────────────────
	cd := resp.Header.Get("Content-Disposition")
	if strings.Contains(cd, ".ps1") || strings.Contains(cd, "bootstrap") {
		t.Errorf("[REGRESSION] Content-Disposition = %q contains '.ps1' or 'bootstrap'.\n"+
			"Fix: GET /downloads/installer in registerRoutes() must call handleDownloadInstaller, NOT handleBootstrapScript.", cd)
	}
	if !strings.Contains(strings.ToLower(cd), ".exe") {
		t.Errorf("[REGRESSION] Content-Disposition = %q does not contain '.exe'", cd)
	}
}

// TestHandleDownloadInstaller_NoInstallerReturns404 verifies that a missing
// installer returns 404 with a helpful message — NOT a bootstrap script body.
func TestHandleDownloadInstaller_NoInstallerReturns404(t *testing.T) {
	t.Setenv("ARCVAULT_VERSION", "v0.5.1")

	srv := newTestServer(t, &config.Config{
		Port:         8080,
		InstallerDir: t.TempDir(), // empty — no exe present
	})

	w := httptest.NewRecorder()
	srv.handleDownloadInstaller(w, httptest.NewRequest(http.MethodGet, "/downloads/installer", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when installer missing, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "Invoke-WebRequest") || strings.Contains(body, "bootstrap") ||
		strings.Contains(body, "ArcVault-Agent") {
		t.Errorf("[REGRESSION] missing-installer response contains bootstrap script content.\n"+
			"Route is wired to the wrong handler. Body: %s", body)
	}
}

// TestRouteTable_InstallerNotBootstrap calls registerRoutes and verifies that
// a plain (unauthenticated) GET /downloads/installer does NOT return a bootstrap
// script body. (With no auth it should return 401/403, not a ps1 payload.)
func TestRouteTable_InstallerNotBootstrap(t *testing.T) {
	srv := newTestServer(t, &config.Config{
		Port:       8080,
		AdminToken: "test-token",
	})
	srv.registerRoutes()

	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/downloads/installer", nil))

	body := w.Body.String()
	// If the route is wired to handleBootstrapScript it will try to read a cert
	// and return an error about certs — or in a broken state dump script content.
	// Either way, it must NOT contain bootstrap script markers.
	if strings.Contains(body, "Invoke-WebRequest") || strings.Contains(body, "ArcVault-Agent") {
		t.Errorf("[REGRESSION] Unauthenticated /downloads/installer returned bootstrap script content.\n"+
			"registerRoutes() has the wrong handler on this route.\nBody: %s", body)
	}
	// Should be auth-rejected, not 200.
	if w.Code == http.StatusOK {
		t.Errorf("[REGRESSION] /downloads/installer returned 200 with no auth — route is not protected.")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// REGRESSION: Service start — run-service arg must be present in registration
// ─────────────────────────────────────────────────────────────────────────────

// TestHandleVersion_ReportsBuiltInVersion verifies that /api/version returns the
// Version string injected at build time, not a blank or hardcoded fallback like "2.0".
func TestHandleVersion_ReportsBuiltInVersion(t *testing.T) {
	if Version == "" {
		t.Error("[REGRESSION] server.Version is empty — ldflags were not applied at build time.\n" +
			"Fix: go build -ldflags \"-X arcvault/coordinator/server.Version=vX.Y.Z\"")
	}
	if Version == "2.0" || Version == "v2.0" {
		t.Errorf("[REGRESSION] server.Version = %q — wrong version baked in.\n"+
			"Coordinator must NOT be built with the project folder name as the version.\n"+
			"Fix: rebuild with -ldflags \"-X main.Version=$(cat VERSION)\"", Version)
	}
}
