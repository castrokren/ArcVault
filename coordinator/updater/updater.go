package updater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type UpdateInfo struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url"`
	AssetURL        string `json:"asset_url"`
}

type ProgressEvent struct {
	Type    string `json:"type"`
	Step    string `json:"step"`
	Pct     int    `json:"pct"`
	Message string `json:"message"`
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
	HTMLURL string `json:"html_url"`
}

// CheckLatestRelease fetches the latest release from GitHub API.
func CheckLatestRelease(currentVersion string) (*UpdateInfo, error) {
	resp, err := http.Get("https://api.github.com/repos/castrokren/ArcVault/releases/latest")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release JSON: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	assetURL, err := resolveAssetURL(release.Assets)
	if err != nil {
		return nil, err
	}

	updateAvailable := compareVersions(currentVersion, latestVersion) < 0

	return &UpdateInfo{
		Current:         currentVersion,
		Latest:          latestVersion,
		UpdateAvailable: updateAvailable,
		ReleaseURL:      release.HTMLURL,
		AssetURL:        assetURL,
	}, nil
}

// resolveAssetURL finds the appropriate asset URL for the current platform from GitHub release.
func resolveAssetURL(assets []struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	assetName := fmt.Sprintf("coordinator_%s_%s", goos, goarch)
	if goos == "windows" {
		assetName += ".exe"
	}

	for _, asset := range assets {
		if asset.Name == assetName {
			return asset.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("no release asset found for your platform (%s/%s)", goos, goarch)
}

// ResolveAssetURL finds the appropriate asset URL for the current platform (for testing).
func ResolveAssetURL(assets []struct {
	Name        string
	DownloadURL string
}) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	assetName := fmt.Sprintf("coordinator_%s_%s", goos, goarch)
	if goos == "windows" {
		assetName += ".exe"
	}

	for _, asset := range assets {
		if asset.Name == assetName {
			return asset.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("no release asset found for your platform (%s/%s)", goos, goarch)
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

// ExecuteUpdate handles the full update flow, including service vs. terminal mode.
func ExecuteUpdate(stagedPath, currentPath string, progress func(ProgressEvent)) error {
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
