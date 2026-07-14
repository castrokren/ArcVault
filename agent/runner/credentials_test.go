package runner

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyCredentials_NilCredentials(t *testing.T) {
	job := Job{
		ID:          "job-1",
		Credentials: nil,
	}

	cleanup, err := applyCredentials(job)
	assert.NoError(t, err)
	assert.NotNil(t, cleanup)

	// Cleanup should be callable without panic
	cleanup()
}

func TestApplyCredentials_UnknownType(t *testing.T) {
	job := Job{
		ID: "job-1",
		Credentials: &JobCredentials{
			Type: "UNKNOWN",
			Data: map[string]interface{}{},
		},
	}

	cleanup, err := applyCredentials(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown credential type")
	// Cleanup should still be callable
	cleanup()
}

func TestApplySSHKeyCredentials(t *testing.T) {
	keyData := "-----BEGIN OPENSSH PRIVATE KEY-----\nMIIEpAIBAAKCAQEA0z7T..."

	cleanup, err := applySSHKeyCredentials(keyData)
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	// Verify SSH_KEY_PATH env var is set
	keyPath := os.Getenv("SSH_KEY_PATH")
	assert.NotEmpty(t, keyPath)

	// Verify file exists
	fileInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.False(t, fileInfo.IsDir())

	// Verify file content
	content, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, keyData, string(content))

	// Call cleanup
	cleanup()

	// Verify file is deleted
	_, err = os.Stat(keyPath)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestApplySSHKeyCredentials_MissingKey(t *testing.T) {
	job := Job{
		ID: "job-1",
		Credentials: &JobCredentials{
			Type: "SSH",
			Data: map[string]interface{}{},
		},
	}

	cleanup, err := applyCredentials(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must include either 'key' or 'password'")
	// Cleanup should still be callable
	cleanup()
}

func TestApplySSHPasswordCredentials(t *testing.T) {
	// Skip if sshpass is not installed
	if _, err := exec.LookPath("sshpass"); err != nil {
		t.Skip("sshpass not found, skipping password auth test")
	}

	password := "test-password-123"
	cleanup, err := applySSHPasswordCredentials(password)
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	// Verify SSHPASS env var is set
	envPassword := os.Getenv("SSHPASS")
	assert.Equal(t, password, envPassword)

	// Call cleanup
	cleanup()

	// Verify env var is unset
	envPassword = os.Getenv("SSHPASS")
	assert.Empty(t, envPassword)
}

func TestApplySSHCredentials_PreferKeyOverPassword(t *testing.T) {
	keyData := "-----BEGIN OPENSSH PRIVATE KEY-----\nkey-content..."
	password := "test-password"

	job := Job{
		ID: "job-1",
		Credentials: &JobCredentials{
			Type: "SSH",
			Data: map[string]interface{}{
				"key":      keyData,
				"password": password,
			},
		},
	}

	cleanup, err := applySSHCredentials(job.Credentials.Data)
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	// Should have created a key file
	keyPath := os.Getenv("SSH_KEY_PATH")
	assert.NotEmpty(t, keyPath)

	// Password env var should NOT be set (key takes precedence)
	envPassword := os.Getenv("SSHPASS")
	assert.Empty(t, envPassword)

	cleanup()
}

func TestApplyCredentials_SMBMissingFields(t *testing.T) {
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "missing username",
			data: map[string]interface{}{
				"password": "pass",
				"host":     "host",
			},
		},
		{
			name: "missing password",
			data: map[string]interface{}{
				"username": "user",
				"host":     "host",
			},
		},
		{
			name: "missing host",
			data: map[string]interface{}{
				"username": "user",
				"password": "pass",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			job := Job{
				ID: "job-1",
				Credentials: &JobCredentials{
					Type: "SMB",
					Data: tc.data,
				},
			}

			cleanup, err := applyCredentials(job)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "SMB credentials missing or invalid")
			cleanup()
		})
	}
}

// TestValidateCredField_hostRejectsBadChars ensures the host field strictly
// rejects anything outside alphanumeric, dots, hyphens, underscores.
func TestValidateCredField_hostRejectsBadChars(t *testing.T) {
	err := validateCredField("evil;host", "host")
	if err == nil {
		t.Fatal("host with semicolon must be rejected")
	}
	err2 := validateCredField("host with spaces", "host")
	if err2 == nil {
		t.Fatal("host with spaces must be rejected")
	}
	// Valid host must still pass
	if err := validateCredField("valid-host.local", "host"); err != nil {
		t.Fatalf("valid host rejected: %v", err)
	}
}

// TestValidateCredField_passwordAcceptsSpecialChars ensures the password field
// accepts spaces, quotes, semicolons, pipes, and ampersands — all of which
// are valid in passwords and not injectable via exec.Command without a shell.
func TestValidateCredField_passwordAcceptsSpecialChars(t *testing.T) {
	password := `P@ss w0rd; "x" & |y`
	if err := validateCredField(password, "password"); err != nil {
		t.Fatalf("password with spaces, quotes, semicolons, pipes, ampersands must be accepted: %v", err)
	}
}

// TestValidateCredField_rejectsControlChars ensures all fields reject control
// characters that could corrupt argv boundaries.
func TestValidateCredField_rejectsControlChars(t *testing.T) {
	tests := []struct {
		name  string
		field string
	}{
		{"password with newline", "pass\nword"},
		{"password with NUL", "pass\x00word"},
		{"username with newline", "user\nname"},
		{"username with NUL", "user\x00name"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredField(tc.field, "password")
			if err == nil {
				t.Errorf("expected error for control char in %q", tc.field)
			}
		})
	}
}

func TestApplyCredentials_CredentialsContextRestored(t *testing.T) {
	// Set original SSH_KEY_PATH
	originalPath := "/original/path/to/key"
	os.Setenv("SSH_KEY_PATH", originalPath)
	defer os.Unsetenv("SSH_KEY_PATH")

	keyData := "test-key-data"
	cleanup, err := applySSHKeyCredentials(keyData)
	require.NoError(t, err)

	// Path should be changed
	currentPath := os.Getenv("SSH_KEY_PATH")
	assert.NotEqual(t, originalPath, currentPath)

	// Cleanup should restore original
	cleanup()
	restoredPath := os.Getenv("SSH_KEY_PATH")
	assert.Equal(t, originalPath, restoredPath)
}
