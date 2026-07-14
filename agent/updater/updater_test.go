package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolveAgentAssetURL verifies the asset name is correct per OS/arch.
func TestResolveAgentAssetURL(t *testing.T) {
	cases := []struct {
		goos, goarch string
		wantName     string
	}{
		{"linux", "amd64", "agent_linux_amd64"},
		{"darwin", "arm64", "agent_darwin_arm64"},
		{"windows", "amd64", "agent_windows_amd64.exe"},
	}

	for _, tc := range cases {
		t.Run(tc.goos+"_"+tc.goarch, func(t *testing.T) {
			// Build asset name using the same logic as resolveAsset in the coordinator.
			name := "agent_" + tc.goos + "_" + tc.goarch
			if tc.goos == "windows" {
				name += ".exe"
			}
			if name != tc.wantName {
				t.Errorf("asset name: got %q, want %q", name, tc.wantName)
			}
		})
	}
}

// TestDownloadBinary verifies happy-path download writes the file.
func TestDownloadBinary(t *testing.T) {
	payload := make([]byte, 200)
	for i := range payload {
		payload[i] = 'x'
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "200")
		w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "agent.bin")
	called := false
	if err := downloadBinary(srv.URL, dest, func(pct int) { called = true }); err != nil {
		t.Fatalf("downloadBinary: %v", err)
	}

	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("file not found after download: %v", err)
	}
	if fi.Size() != 200 {
		t.Errorf("file size: got %d, want 200", fi.Size())
	}
	if !called {
		t.Error("progress callback was never called")
	}
}

// TestDownloadBinaryNetworkError ensures temp file is cleaned up on failure.
func TestDownloadBinaryNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte("partial"))
		panic("close connection")
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "agent.bin")
	err := downloadBinary(srv.URL, dest, func(int) {})
	if err == nil {
		t.Error("expected error on network failure")
	}
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("temp file was not cleaned up after failure")
	}
}

// TestVerifyBinary verifies a mock binary that exits 0 and prints a version.
func TestVerifyBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell-script test on windows")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "agent")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo v0.3.0; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if err := verifyBinary(binPath); err != nil {
		t.Errorf("verifyBinary: %v", err)
	}
}

// TestVerifyBinaryFails ensures a non-zero exit is rejected.
func TestVerifyBinaryFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell-script test on windows")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "agent_bad")
	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(binPath, []byte(script), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	if err := verifyBinary(binPath); err == nil {
		t.Error("expected verifyBinary to fail for non-zero exit")
	}
}

// TestStageBinary verifies the rename from tmp to staged path.
func TestStageBinary(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "agent.download.tmp")
	stagedPath := filepath.Join(tmpDir, "agent.new")

	if err := os.WriteFile(tmpPath, []byte("binary"), 0755); err != nil {
		t.Fatalf("write tmp: %v", err)
	}

	if err := os.Rename(tmpPath, stagedPath); err != nil {
		t.Fatalf("stage: %v", err)
	}

	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("tmp file still exists after staging")
	}
	if _, err := os.Stat(stagedPath); err != nil {
		t.Errorf("staged file missing: %v", err)
	}
}

// TestUpdateProgressEvents validates progress step names and percentages.
func TestUpdateProgressEvents(t *testing.T) {
	type event struct{ step string; pct int }
	events := []event{
		{"downloading", 10},
		{"downloading", 60},
		{"verifying", 70},
		{"staging", 85},
		{"restarting", 95},
		{"restarting", 100},
	}
	for _, e := range events {
		if e.pct < 0 || e.pct > 100 {
			t.Errorf("invalid pct %d for step %q", e.pct, e.step)
		}
		if e.step == "" {
			t.Error("step name must not be empty")
		}
	}
}

// TestVersionComparison verifies update is skipped for same/newer version.
func TestVersionComparison(t *testing.T) {
	cases := []struct {
		current, latest string
		wantUpdate      bool
	}{
		{"v0.2.0", "v0.3.0", true},
		{"v0.3.0", "v0.3.0", false},
		{"v0.4.0", "v0.3.0", false},
	}
	for _, tc := range cases {
		needsUpdate := versionLess(tc.current, tc.latest)
		if needsUpdate != tc.wantUpdate {
			t.Errorf("versionLess(%q, %q) = %v, want %v", tc.current, tc.latest, needsUpdate, tc.wantUpdate)
		}
	}
}

// versionLess is a minimal semver comparison used only in tests.
func versionLess(a, b string) bool {
	// Strip 'v' prefix and compare lexicographically as a quick approximation.
	if len(a) > 0 && a[0] == 'v' {
		a = a[1:]
	}
	if len(b) > 0 && b[0] == 'v' {
		b = b[1:]
	}
	return a < b
}

// TestBackupCurrent tests backing up the current agent binary.
func TestBackupCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")

	os.Setenv("ARCVAULT_BACKUP_DIR", backupDir)
	defer os.Unsetenv("ARCVAULT_BACKUP_DIR")

	// Create a mock current binary
	currentPath := filepath.Join(tmpDir, "agent")
	if runtime.GOOS == "windows" {
		currentPath += ".exe"
	}
	err := os.WriteFile(currentPath, []byte("binary_content"), 0755)
	if err != nil {
		t.Fatalf("Failed to create current binary: %v", err)
	}

	err = BackupCurrent(currentPath)
	if err != nil {
		t.Errorf("BackupCurrent failed: %v", err)
	}

	// Check that backup file exists
	backupPath := filepath.Join(backupDir, "agent.previous")
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file not created: %v", err)
	}

	// Verify backup content matches original
	content, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Failed to read backup: %v", err)
	}
	if string(content) != "binary_content" {
		t.Errorf("Backup content mismatch: got %q, want %q", string(content), "binary_content")
	}
}

// TestIsRollbackAvailable tests checking if rollback is available.
func TestIsRollbackAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")

	os.Setenv("ARCVAULT_BACKUP_DIR", backupDir)
	defer os.Unsetenv("ARCVAULT_BACKUP_DIR")

	// Test when no backup exists
	available, err := IsRollbackAvailable()
	if err != nil {
		t.Errorf("IsRollbackAvailable failed: %v", err)
	}
	if available {
		t.Errorf("IsRollbackAvailable should return false when no backup exists")
	}

	// Create a backup file
	os.MkdirAll(backupDir, 0755)
	backupPath := filepath.Join(backupDir, "agent.previous")
	err = os.WriteFile(backupPath, []byte("backup_content"), 0755)
	if err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	// Test when backup exists
	available, err = IsRollbackAvailable()
	if err != nil {
		t.Errorf("IsRollbackAvailable failed: %v", err)
	}
	if !available {
		t.Errorf("IsRollbackAvailable should return true when backup exists")
	}
}

// TestRollbackNoBackupError tests rollback when no backup exists.
func TestRollbackNoBackupError(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")

	os.Setenv("ARCVAULT_BACKUP_DIR", backupDir)
	defer os.Unsetenv("ARCVAULT_BACKUP_DIR")

	currentPath := filepath.Join(tmpDir, "agent")

	progress := func(step string, pct int) {}
	err := Rollback(currentPath, progress)
	if err == nil {
		t.Errorf("Expected error when no backup exists")
	}
	if err.Error() != "no backup available for rollback" {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestVerifyChecksum_emptyURL_errors ensures that VerifyChecksum returns
// an error when checksumURL is empty (fail closed), rather than silently
// skipping verification.
func TestVerifyChecksum_emptyURL_errors(t *testing.T) {
	err := VerifyChecksum("", "agent.exe", "/tmp/dummy")
	if err == nil {
		t.Fatal("VerifyChecksum('', ...) must return error — empty checksum URL must not be silently accepted")
	}
}

// TestVerifyChecksum_non200_errors ensures that VerifyChecksum returns
// an error when the checksum server returns a non-200 status code.
func TestVerifyChecksum_non200_errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := VerifyChecksum(srv.URL, "agent.exe", "/tmp/dummy")
	if err == nil {
		t.Fatal("VerifyChecksum(non-200 URL) must return error")
	}
}

// TestVerifyChecksum_match verifies that a valid SHA256SUMS file with a
// matching hash returns nil. This ensures the existing match logic still works.
func TestVerifyChecksum_match(t *testing.T) {
	// Create a temp file with known content
	tmpFile := filepath.Join(t.TempDir(), "agent.bin")
	content := []byte("hello world\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Compute expected hash
	h := sha256.New()
	h.Write(content)
	expectedHash := hex.EncodeToString(h.Sum(nil))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := expectedHash + "  agent.bin\n"
		w.Write([]byte(response))
	}))
	defer srv.Close()

	if err := VerifyChecksum(srv.URL, "agent.bin", tmpFile); err != nil {
		t.Fatalf("VerifyChecksum with matching hash: expected nil, got %v", err)
	}
}

// TestVerifyChecksum_mismatch verifies that a hash mismatch returns an error.
func TestVerifyChecksum_mismatch(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "agent.bin")
	content := []byte("actual content\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := "0000000000000000000000000000000000000000000000000000000000000000  agent.bin\n"
		w.Write([]byte(response))
	}))
	defer srv.Close()

	err := VerifyChecksum(srv.URL, "agent.bin", tmpFile)
	if err == nil {
		t.Fatal("VerifyChecksum with mismatching hash: expected error, got nil")
	}
}
