package credcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrKeyNotSet = errors.New("encryption key not set in environment (ARCVAULT_CREDENTIAL_KEY)")
	ErrKeyInvalid = errors.New("encryption key invalid (must be 32 bytes / 64 hex chars)")
)

// LoadKey reads the encryption key from the environment variable.
// Returns ErrKeyNotSet if the env var is not set.
// Returns ErrKeyInvalid if the key is not exactly 32 bytes.
func LoadKey() ([]byte, error) {
	keyHex := os.Getenv("ARCVAULT_CREDENTIAL_KEY")
	if keyHex == "" {
		return nil, ErrKeyNotSet
	}
	return LoadKeyFromString(keyHex)
}

// LoadKeyFromString decodes a hex-encoded 32-byte key from a string value.
// Use this when the key comes from config rather than the environment.
func LoadKeyFromString(keyHex string) ([]byte, error) {
	if keyHex == "" {
		return nil, ErrKeyNotSet
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, ErrKeyInvalid
	}

	if len(key) != 32 {
		return nil, ErrKeyInvalid
	}

	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns a nonce (12 bytes) + ciphertext + tag format.
func Encrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
// Expects nonce + ciphertext + tag format (output of Encrypt).
func Decrypt(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
