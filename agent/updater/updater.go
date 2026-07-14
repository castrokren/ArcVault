package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// HandleUpdateCommand runs the full update sequence: download → verify → stage → apply.
// progressFn is called at each step with a step name and percentage (0–100).
// Golden rule: the running binary is never modified before staging succeeds.
func HandleUpdateCommand(version, url, checksumURL string, progressFn func(step string, pct int)) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	exeDir := filepath.Dir(exePath)
	tmpPath := filepath.Join(exeDir, "agent.download.tmp")
	stagedPath := filepath.Join(exeDir, "agent.new")

	progressFn("downloading", 10)
	if err := downloadBinary(url, tmpPath, func(pct int) {
		// Scale download progress to 10–60%.
		progressFn("downloading", 10+pct*50/100)
	}); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download failed: %w", err)
	}

	progressFn("verifying", 65)
	assetName := path.Base(url)
	if err := VerifyChecksum(checksumURL, assetName, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	progressFn("verifying", 75)
	if err := verifyBinary(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("verification failed: %w", err)
	}

	progressFn("staging", 85)
	if err := os.Rename(tmpPath, stagedPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("staging failed: %w", err)
	}

	// Backup current binary before swap
	progressFn("backup", 90)
	if err := BackupCurrent(exePath); err != nil {
		os.Remove(stagedPath)
		return fmt.Errorf("backup failed: %w", err)
	}

	// Beyond this point the staged binary exists; apply is platform-specific.
	progressFn("restarting", 95)
	if err := ApplyUpdate(stagedPath, exePath); err != nil {
		os.Remove(stagedPath)
		return fmt.Errorf("apply failed: %w", err)
	}

	progressFn("restarting", 100)
	return nil
}

func downloadBinary(url, destPath string, progress func(pct int)) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = 1
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 32*1024)
	downloaded := int64(0)
	lastReported := 0

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				os.Remove(destPath)
				return fmt.Errorf("write failed: %w", werr)
			}
			downloaded += int64(n)
			pct := int(downloaded * 100 / totalSize)
			if pct > lastReported && pct%5 == 0 {
				progress(pct)
				lastReported = pct
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

func verifyBinary(path string) error {
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			return fmt.Errorf("chmod failed: %w", err)
		}
	}
	cmd := exec.Command(path, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("binary --version check failed: %w", err)
	}
	return nil
}

// VerifyChecksum downloads the SHA256SUMS file from checksumURL and verifies
// that the file at localPath matches the expected hash for assetName.
// Returns an error if checksumURL is empty, HTTP status is not 200,
// the asset is missing from the checksum file, or the hash does not match.
func VerifyChecksum(checksumURL, assetName, localPath string) error {
	if checksumURL == "" {
		return fmt.Errorf("checksum URL is empty — cannot verify download integrity")
	}

	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

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

// BackupCurrent copies the current agent binary to the backup directory before update.
func BackupCurrent(currentPath string) error {
	backupDir, err := getBackupDir()
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "agent.previous")
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
	backupPath := filepath.Join(backupDir, "agent.previous")
	_, err = os.Stat(backupPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Rollback restores the previous agent binary from backup.
func Rollback(currentPath string, progressFn func(step string, pct int)) error {
	backupDir, err := getBackupDir()
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, "agent.previous")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup available for rollback")
	}

	progressFn("verify_backup", 10)
	if err := verifyBinary(backupPath); err != nil {
		return fmt.Errorf("backup binary is corrupt or invalid: %w", err)
	}

	exeDir := filepath.Dir(currentPath)
	stagedPath := filepath.Join(exeDir, "agent.rollback")

	progressFn("stage_backup", 30)
	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup for rollback: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(stagedPath)
	if err != nil {
		return fmt.Errorf("failed to stage backup: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(stagedPath)
		return fmt.Errorf("failed to copy backup to stage: %w", err)
	}

	progressFn("apply_rollback", 60)
	if err := ApplyUpdate(stagedPath, currentPath); err != nil {
		os.Remove(stagedPath)
		return fmt.Errorf("apply failed: %w", err)
	}

	progressFn("done", 100)
	return nil
}
