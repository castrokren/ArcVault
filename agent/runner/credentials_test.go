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
