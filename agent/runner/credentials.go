package runner

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// applyCredentials sets up credentials for a job based on its type.
// Returns a cleanup function (to be deferred) and any error.
// If job has no credentials, returns a no-op cleanup function and nil error.
func applyCredentials(job Job) (func(), error) {
	if job.Credentials == nil {
		return func() {}, nil
	}

	credType := job.Credentials.Type
	data := job.Credentials.Data

	switch credType {
	case "SMB":
		return applySMBCredentials(data)
	case "SSH":
		return applySSHCredentials(data)
	default:
		return func() {}, fmt.Errorf("unknown credential type: %s", credType)
	}
}

// applySMBCredentials sets up SMB credentials using Windows cmdkey utility.
// Only works on Windows.
func applySMBCredentials(data map[string]interface{}) (func(), error) {
	if runtime.GOOS != "windows" {
		return func() {}, fmt.Errorf("SMB credentials only supported on Windows")
	}

	// Extract fields
	username, ok := data["username"].(string)
	if !ok || username == "" {
		return func() {}, fmt.Errorf("SMB credentials missing or invalid username")
	}

	password, ok := data["password"].(string)
	if !ok || password == "" {
		return func() {}, fmt.Errorf("SMB credentials missing or invalid password")
	}

	host, ok := data["host"].(string)
	if !ok || host == "" {
		return func() {}, fmt.Errorf("SMB credentials missing or invalid host")
	}

	// Use cmdkey to store credentials
	// Format: cmdkey /add:hostname /user:username /pass:password
	cmd := exec.Command("cmdkey", "/add:"+host, "/user:"+username, "/pass:"+password)
	if err := cmd.Run(); err != nil {
		return func() {}, fmt.Errorf("failed to set SMB credentials with cmdkey: %w", err)
	}

	// Cleanup: remove credentials after job completes
	cleanup := func() {
		cmd := exec.Command("cmdkey", "/delete:"+host)
		cmd.Run() // ignore errors in cleanup
	}

	return cleanup, nil
}

// applySSHCredentials sets up SSH credentials (either key-based or password-based).
// For key-based: creates temp key file, sets SSH_KEY_PATH env var
// For password-based: sets SSH_PASSWORD env var (sshpass required)
func applySSHCredentials(data map[string]interface{}) (func(), error) {
	// Check if key-based auth (priority over password)
	if keyData, ok := data["key"].(string); ok && keyData != "" {
		return applySSHKeyCredentials(keyData)
	}

	// Check for password-based auth
	if password, ok := data["password"].(string); ok && password != "" {
		return applySSHPasswordCredentials(password)
	}

	return func() {}, fmt.Errorf("SSH credentials must include either 'key' or 'password'")
}

// applySSHKeyCredentials creates a temporary SSH key file and sets environment variable.
func applySSHKeyCredentials(keyData string) (func(), error) {
	// Create temp file for SSH key
	tempFile, err := os.CreateTemp("", "ssh-key-*.pem")
	if err != nil {
		return func() {}, fmt.Errorf("failed to create temp SSH key file: %w", err)
	}

	// Write key data
	if _, err := tempFile.WriteString(keyData); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return func() {}, fmt.Errorf("failed to write SSH key: %w", err)
	}

	// Set file permissions to 600 (required for SSH keys)
	if err := os.Chmod(tempFile.Name(), 0600); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return func() {}, fmt.Errorf("failed to set SSH key permissions: %w", err)
	}

	tempFile.Close()

	// Set environment variable for SSH agent to use this key
	oldValue := os.Getenv("SSH_KEY_PATH")
	os.Setenv("SSH_KEY_PATH", tempFile.Name())

	// Cleanup: restore env var and delete temp file
	cleanup := func() {
		if oldValue == "" {
			os.Unsetenv("SSH_KEY_PATH")
		} else {
			os.Setenv("SSH_KEY_PATH", oldValue)
		}
		os.Remove(tempFile.Name())
	}

	return cleanup, nil
}

// applySSHPasswordCredentials sets environment variable for sshpass.
func applySSHPasswordCredentials(password string) (func(), error) {
	// Check if sshpass is available (for password auth)
	if _, err := exec.LookPath("sshpass"); err != nil {
		return func() {}, fmt.Errorf("sshpass not found, required for SSH password authentication")
	}

	// Set environment variable for sshpass
	oldValue := os.Getenv("SSHPASS")
	os.Setenv("SSHPASS", password)

	// Cleanup: restore env var
	cleanup := func() {
		if oldValue == "" {
			os.Unsetenv("SSHPASS")
		} else {
			os.Setenv("SSHPASS", oldValue)
		}
	}

	return cleanup, nil
}
