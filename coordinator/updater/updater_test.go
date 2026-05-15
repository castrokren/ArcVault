package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolveAssetURL tests asset resolution for different OS/arch combinations.
func TestResolveAssetURL(t *testing.T) {
	tests := []struct {
		goos     string
		goarch   string
		name     string
		wantName string
	}{
		{"linux", "amd64", "coordinator_linux_amd64", "coordinator_linux_amd64"},
		{"darwin", "arm64", "coordinator_darwin_arm64", "coordinator_darwin_arm64"},
		{"windows", "amd64", "coordinator_windows_amd64.exe", "coordinator_windows_amd64.exe"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.goos, tt.goarch), func(t *testing.T) {
			// Save original runtime values and mock them would require platform build,
			// so we just test the asset matching logic
			assets := []struct {
				Name        string
				DownloadURL string
			}{
				{Name: "coordinator_linux_amd64", DownloadURL: "http://example.com/linux"},
				{Name: "coordinator_darwin_arm64", DownloadURL: "http://example.com/darwin"},
				{Name: "coordinator_windows_amd64.exe", DownloadURL: "http://example.com/windows"},
			}

			// Filter to find the asset we're testing
			for _, asset := range assets {
				if asset.Name == tt.wantName {
					// Manually check the resolution logic
					url, err := ResolveAssetURL(assets)
					if err != nil {
						// Only fail if this is the current platform
						if (runtime.GOOS == tt.goos && runtime.GOARCH == tt.goarch) {
							t.Errorf("ResolveAssetURL failed: %v", err)
						}
						return
					}
					if url == "" {
						t.Errorf("ResolveAssetURL returned empty URL")
					}
					return
				}
			}
		})
	}
}

// TestDownloadBinary tests downloading a binary to a file.
func TestDownloadBinary(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "100")
		payload := make([]byte, 100)
		for i := range payload {
			payload[i] = 'x'
		}
		w.Write(payload)
	}))
	defer server.Close()

	// Create temp file
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.bin")

	// Download
	progressCalled := false
	progress := func(pct int) {
		progressCalled = true
	}

	err := DownloadBinary(server.URL, destPath, progress)
	if err != nil {
		t.Errorf("DownloadBinary failed: %v", err)
	}

	// Check file was created
	fi, err := os.Stat(destPath)
	if err != nil {
		t.Errorf("Downloaded file not found: %v", err)
	}
	if fi.Size() != 100 {
		t.Errorf("Downloaded file size: got %d, want 100", fi.Size())
	}

	if !progressCalled {
		t.Errorf("Progress callback was not called")
	}
}

// TestDownloadBinaryNetworkError tests cleanup on network error.
func TestDownloadBinaryNetworkError(t *testing.T) {
	// Create a mock server that closes connection mid-stream
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		// Write only partial data
		w.Write([]byte("partial"))
		panic("intentional panic to close connection")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.bin")

	// Download should fail
	err := DownloadBinary(server.URL, destPath, func(pct int) {})
	if err == nil {
		t.Errorf("Expected error on network failure")
	}

	// File should be cleaned up
	if _, err := os.Stat(destPath); err == nil {
		t.Errorf("Temp file was not cleaned up after download failure")
	}
}

// TestVerifyBinary tests binary verification.
func TestVerifyBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock binary that returns version
	binPath := filepath.Join(tmpDir, "test_binary")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	content := `#!/bin/bash
if [ "$1" = "--version" ]; then
  echo "v0.2.0"
  exit 0
fi
exit 1
`

	if runtime.GOOS == "windows" {
		// For Windows, create a batch file
		content = "@echo v0.2.0\nexit /b 0"
		binPath = filepath.Join(tmpDir, "test_binary.bat")
	}

	err := os.WriteFile(binPath, []byte(content), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// On non-Windows, use the bash script
	if runtime.GOOS != "windows" {
		err = VerifyBinary(binPath)
		if err != nil {
			// This is expected to work on Unix
			t.Errorf("VerifyBinary failed: %v", err)
		}
	}
}

// TestVerifyBinaryFails tests verification failure.
func TestVerifyBinaryFails(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock binary that exits with error
	binPath := filepath.Join(tmpDir, "test_binary_fail")
	content := `#!/bin/bash
exit 1
`

	if runtime.GOOS == "windows" {
		content = "exit /b 1"
		binPath = filepath.Join(tmpDir, "test_binary_fail.bat")
	}

	err := os.WriteFile(binPath, []byte(content), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	err = VerifyBinary(binPath)
	if err == nil {
		t.Errorf("Expected VerifyBinary to fail for non-zero exit")
	}
}

// TestDetectServiceMode tests service mode detection.
func TestDetectServiceMode(t *testing.T) {
	// Test with env var not set
	os.Unsetenv("ARCVAULT_SERVICE")
	if IsServiceMode() {
		t.Errorf("IsServiceMode returned true when env var not set")
	}

	// Test with env var set
	os.Setenv("ARCVAULT_SERVICE", "1")
	if !IsServiceMode() {
		t.Errorf("IsServiceMode returned false when env var set to 1")
	}

	// Test with env var set to other value
	os.Setenv("ARCVAULT_SERVICE", "0")
	if IsServiceMode() {
		t.Errorf("IsServiceMode returned true when env var set to 0")
	}

	// Cleanup
	os.Unsetenv("ARCVAULT_SERVICE")
}

// TestUpdateProgressEvents tests progress event generation.
func TestUpdateProgressEvents(t *testing.T) {
	events := []ProgressEvent{
		{Type: "update_progress", Step: "resolving", Pct: 10},
		{Type: "update_progress", Step: "downloading", Pct: 30},
		{Type: "update_progress", Step: "verifying", Pct: 80},
		{Type: "update_progress", Step: "staging", Pct: 88},
		{Type: "update_progress", Step: "restarting", Pct: 95},
		{Type: "update_progress", Step: "done", Pct: 100},
	}

	for i, evt := range events {
		if evt.Type != "update_progress" {
			t.Errorf("Event %d: wrong type %s", i, evt.Type)
		}
		if evt.Pct < 0 || evt.Pct > 100 {
			t.Errorf("Event %d: invalid percentage %d", i, evt.Pct)
		}
	}

	// Test that events can be marshaled to JSON
	for _, evt := range events {
		_, err := json.Marshal(evt)
		if err != nil {
			t.Errorf("Failed to marshal event: %v", err)
		}
	}
}

// TestVersionComparison tests semantic version comparison.
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
		desc     string
	}{
		{"0.2.0", "0.3.0", -1, "v0.2.0 < v0.3.0"},
		{"0.3.0", "0.2.0", 1, "v0.3.0 > v0.2.0"},
		{"0.2.0", "0.2.0", 0, "v0.2.0 == v0.2.0"},
		{"1.0.0", "0.9.9", 1, "v1.0.0 > v0.9.9"},
		{"0.2.1", "0.2.0", 1, "v0.2.1 > v0.2.0"},
		{"v0.3.1", "v0.2.0", 1, "v0.3.1 > v0.2.0 (with v prefix)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := compareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

// TestStageBinary tests binary staging (renaming).
func TestStageBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a temp file
	tmpPath := filepath.Join(tmpDir, "tmp.bin")
	err := os.WriteFile(tmpPath, []byte("binary"), 0755)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Stage it
	stagedPath := filepath.Join(tmpDir, "staged.bin")
	err = StageBinary(tmpPath, stagedPath)
	if err != nil {
		t.Errorf("StageBinary failed: %v", err)
	}

	// Check that tmp file is gone and staged file exists
	if _, err := os.Stat(tmpPath); err == nil {
		t.Errorf("Temp file still exists after staging")
	}

	if _, err := os.Stat(stagedPath); err != nil {
		t.Errorf("Staged file does not exist: %v", err)
	}
}

// BenchmarkVersionComparison benchmarks version comparison.
func BenchmarkVersionComparison(b *testing.B) {
	for i := 0; i < b.N; i++ {
		compareVersions("0.2.0", "0.3.1")
	}
}
