package server

import (
	"encoding/json"
	"testing"

	"arcvault/coordinator/config"
	"arcvault/coordinator/internal/credcrypto"
)

// 32-byte keys as 64 hex chars.
const (
	keyA = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	keyB = "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"
)

// decryptCredentials must return credentials on the right key and error (not a
// silent nil) on a wrong/missing key or a missing profile — otherwise a bound
// job would run a backup with no credentials instead of failing loudly.
func TestDecryptCredentials_LoudOnFailure(t *testing.T) {
	s := newTestServer(t, WithConfig(&config.Config{
		Port:          8080,
		AdminToken:    "test-token",
		JWTSecret:     "test-secret",
		CredentialKey: keyA,
	}))

	key, err := credcrypto.LoadKeyFromString(keyA)
	if err != nil {
		t.Fatalf("load key: %v", err)
	}
	plaintext, _ := json.Marshal(map[string]interface{}{"username": "svc", "password": "hunter2"})
	ct, err := credcrypto.Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := s.db.CreateCredentialProfile("cred-test", "smb", "SMB", ct); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// Right key: creds returned, no error.
	creds, err := s.decryptCredentials("cred-test")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if creds["password"] != "hunter2" {
		t.Fatalf("wrong plaintext: %v", creds)
	}

	// Wrong key: must error, not return nil silently.
	s.cfg.CredentialKey = keyB
	if _, err := s.decryptCredentials("cred-test"); err == nil {
		t.Fatal("expected error on wrong key, got nil (silent failure)")
	}

	// Missing profile: must error.
	s.cfg.CredentialKey = keyA
	if _, err := s.decryptCredentials("cred-nope"); err == nil {
		t.Fatal("expected error on missing profile, got nil")
	}
}
