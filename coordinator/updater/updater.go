package updater

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type UpdateInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	AssetURL        string `json:"asset_url"`
	ChecksumURL     string `json:"checksum_url"` // URL to SHA256SUMS file
}

type ProgressEvent struct {
	Type    string `json:"type"`
	Step    string `json:"step"`
	Pct     int    `json:"pct"`
	Message string `json:"message"`
}

// ReleaseAsset is a single downloadable asset from a GitHub release.
type ReleaseAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
	HTMLURL string         `json:"html_url"`
}

// CheckLatestRelease fetches the latest release from GitHub API.
func CheckLatestRelease(currentVersion string) (*UpdateInfo, error) {
	resp, err := http.Get("https://api.github.com/repos/castrokren/ArcVault/releases/latest")
	if err != nil {
		log.Printf("GitHub release check failed: %v", err)
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitHub release body read failed: %v", err)
		return nil, fmt.Errorf("failed to read release body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GitHub release check returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		log.Printf("GitHub release JSON parse failed: %v; body=%s", err, string(body))
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	assetURL, err := resolveAssetURL(release.Assets)
	if err != nil {
		log.Printf("GitHub release asset resolution failed: %v; assets=%v", err, releaseAssetsSummary(release.Assets))
		return nil, err
	}

	// Find SHA256SUMS file
	checksumURL := ""
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, "SHA256SUMS") || strings.EqualFold(asset.Name, "sha256sums.txt") {
			checksumURL = asset.DownloadURL
			break
		}
	}

	log.Printf("GitHub release check: current=%s latest=%s tag=%s asset_url=%s assets=%d", currentVersion, latestVersion, release.TagName, assetURL, len(release.Assets))

	updateAvailable := compareVersions(currentVersion, latestVersion) < 0

	return &UpdateInfo{
		Current:         currentVersion,
		Latest:          latestVersion,
		UpdateAvailable: updateAvailable,
		ReleaseURL:      release.HTMLURL,
		AssetURL:        assetURL,
		ChecksumURL:     checksumURL,
	}, nil
}

func releaseAssetsSummary(assets []ReleaseAsset) []string {
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		names = append(names, asset.Name)
	}
	return names
}

// resolveAssetURL finds the coordinator asset URL for the current platform.
func resolveAssetURL(assets []ReleaseAsset) (string, error) {
	return resolveAsset("coordinator", runtime.GOOS, runtime.GOARCH, assets)
}

// ResolveAssetURL finds the coordinator asset URL for the current platform (exported for tests).
func ResolveAssetURL(assets []ReleaseAsset) (string, error) {
	return resolveAsset("coordinator", runtime.GOOS, runtime.GOARCH, assets)
}

// ResolveAgentAssetURL finds the agent asset URL for the given OS/arch.
func ResolveAgentAssetURL(goos, goarch string, assets []ReleaseAsset) (string, error) {
	return resolveAsset("agent", goos, goarch, assets)
}

// resolveAsset finds a release asset URL for the given binary prefix, OS, and arch.
// It supports both exact asset names and versioned/release archive names.
func resolveAsset(prefix, goos, goarch string, assets []ReleaseAsset) (string, error) {
	exact := fmt.Sprintf("%s_%s_%s", prefix, goos, goarch)
	alt := fmt.Sprintf("%s-%s-%s", prefix, goos, goarch)

	// Preferred: platform-specific name (coordinator_windows_amd64.exe)
	for _, asset := range assets {
		name := asset.Name
		if strings.EqualFold(name, exact) ||
			strings.EqualFold(name, exact+".exe") ||
			strings.EqualFold(name, alt) ||
			strings.EqualFold(name, alt+".exe") {
			return asset.DownloadURL, nil
		}
	}

	// Fallback: plain name (coordinator.exe or coordinator)
	plainExe := prefix + ".exe"
	for _, asset := range assets {
		name := asset.Name
		if strings.EqualFold(name, prefix) || strings.EqualFold(name, plainExe) {
			return asset.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("no release asset found for %s on %s/%s", prefix, goos, goarch)
}

// FetchLatestRelease fetches the latest GitHub release assets and release URL.
func FetchLatestRelease() ([]ReleaseAsset, string, error) {
	resp, err := http.Get("https://api.github.com/repos/castrokren/ArcVault/releases/latest")
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, "", fmt.Errorf("failed to parse release JSON: %w", err)
	}

	return release.Assets, release.HTMLURL, nil
}

// DownloadBinary downloads the binary to a temporary file and calls progress callback.
func DownloadBinary(assetURL, destPath string, progress func(pct int)) error {
	resp, err := http.Get(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = 1 // Avoid division by zero
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	buf := make([]byte, 32*1024) // 32KB chunks
	downloaded := int64(0)
	lastReportedPct := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				os.Remove(destPath)
				return fmt.Errorf("failed to write binary: %w", writeErr)
			}
			downloaded += int64(n)

			pct := int((downloaded * 100) / totalSize)
			if pct > lastReportedPct && pct%5 == 0 {
				progress(pct)
				lastReportedPct = pct
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(destPath)
			return fmt.Errorf("download interrupted: %w", err)
		}
	}

	progress(100)
	return nil
}

// VerifyChecksum downloads the SHA256SUMS file from checksumURL and verifies
// that the file at localPath matches the expected hash for assetName.
// Returns nil if the checksum matches or if checksumURL is empty (skips check).
func VerifyChecksum(checksumURL, assetName, localPath string) error {
	if checksumURL == "" {
		log.Printf("Updater: no checksum URL available, skipping integrity check")
		return nil
	}

	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	// Parse "hash  filename" lines
	var expected string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.EqualFold(fields[1], assetName) {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksum for %s not found in SHA256SUMS", assetName)
	}

	// Hash the local file
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to hash file: %w", err)
	}
	actual := hex.EncodeToString(h.Sum(nil))

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}
	log.Printf("Updater: checksum verified OK for %s (%s)", assetName, actual[:12]+"...")
	return nil
}

// VerifyBinary checks that the downloaded binary is executable and returns a version string.
func VerifyBinary(path string) error {
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	cmd := exec.Command(path, "--version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return fmt.Errorf("binary did not return a version string")
	}

	return nil
}

// StageBinary moves the temporary binary to the staging location (atomic on Unix).
func StageBinary(tmpPath, stagedPath string) error {
	if err := os.Rename(tmpPath, stagedPath); err != nil {
		return fmt.Errorf("failed to stage binary: %w", err)
	}
	return nil
}

// IsServiceMode checks if the coordinator is running as a service.
func IsServiceMode() bool {
	return os.Getenv("ARCVAULT_SERVICE") == "1"
}

// getBackupDir returns the platform-specific backup directory path.
// Respects ARCVAULT_BACKUP_DIR env var for testing.
func getBackupDir() (string, error) {
	if testDir := os.Getenv("ARCVAULT_BACKUP_DIR"); testDir != "" {
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create backup directory: %w", err)
		}
		return testDir, nil
	}

	var backupDir string
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = filepath.Join(os.Getenv("SystemDrive"), "ProgramData")
		}
		backupDir = filepath.Join(programData, "ArcVault", "backups")
	default: // linux, darwin
		backupDir = "/var/lib/arcvault/backups"
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}
	return backupDir, nil
}

// BackupCurrent copies the current binary to the backup directory before update.
func BackupCurrent(currentPath string) error {
	backupDir, err := getBackupDir()
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "coordinator.previous")
	src, err := os.Open(currentPath)
	if err != nil {
		return fmt.Errorf("failed to open current binary for backup: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(backupPath)
		return fmt.Errorf("failed to copy binary to backup: %w", err)
	}

	return nil
}

// IsRollbackAvailable checks if a backup binary exists.
func IsRollbackAvailable() (bool, error) {
	backupDir, err := getBackupDir()
	if err != nil {
		return false, err
	}
	backupPath := filepath.Join(backupDir, "coordinator.previous")
	_, err = os.Stat(backupPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Rollback restores the previous binary from backup.
func Rollback(currentPath string, progress func(ProgressEvent)) error {
	backupDir, err := getBackupDir()
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "coordinator.previous")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup available for rollback")
	}

	progress(ProgressEvent{
		Type:    "rollback_progress",
		Step:    "verify_backup",
		Pct:     10,
		Message: "Verifying backup binary...",
	})

	if err := VerifyBinary(backupPath); err != nil {
		return fmt.Errorf("backup binary is corrupt or invalid: %w", err)
	}

	progress(ProgressEvent{
		Type:    "rollback_progress",
		Step:    "stage_backup",
		Pct:     30,
		Message: "Staging backup binary...",
	})

	stagePath := currentPath + ".rollback"
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup for rollback: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(stagePath)
	if err != nil {
		return fmt.Errorf("failed to stage backup: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(stagePath)
		return fmt.Errorf("failed to copy backup to stage: %w", err)
	}

	progress(ProgressEvent{
		Type:    "rollback_progress",
		Step:    "apply_rollback",
		Pct:     60,
		Message: "Applying rollback...",
	})

	return ApplyUpdate(stagePath, currentPath, progress)
}

// ExecuteUpdate handles the full update flow, including service vs. terminal mode.
func ExecuteUpdate(stagedPath, currentPath string, progress func(ProgressEvent)) error {
	progress(ProgressEvent{
		Type:    "update_progress",
		Step:    "backup_current",
		Pct:     80,
		Message: "Creating backup of current binary...",
	})

	if err := BackupCurrent(currentPath); err != nil {
		return err
	}

	if !IsServiceMode() {
		// Terminal mode: just rename and emit done_manual event
		if err := os.Rename(stagedPath, currentPath); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		progress(ProgressEvent{
			Type:    "update_progress",
			Step:    "done_manual",
			Pct:     100,
			Message: "Binary updated. Please restart the coordinator manually.",
		})
		return nil
	}

	// Service mode: delegate to platform-specific handler
	return ApplyUpdate(stagedPath, currentPath, progress)
}

// compareVersions compares two semantic versions (e.g., "0.2.0" vs "0.3.1").
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(strings.TrimPrefix(v1, "v"), ".")
	parts2 := strings.Split(strings.TrimPrefix(v2, "v"), ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}

	return 0
}
