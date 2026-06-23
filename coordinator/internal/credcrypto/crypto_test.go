package credcrypto

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKey_Success(t *testing.T) {
	// 32-byte key encoded as 64-character hex string
	keyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("ARCVAULT_CREDENTIAL_KEY", keyHex)

	key, err := LoadKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestLoadKey_NotSet(t *testing.T) {
	os.Unsetenv("ARCVAULT_CREDENTIAL_KEY")

	_, err := LoadKey()
	assert.Equal(t, ErrKeyNotSet, err)
}

func TestLoadKey_InvalidHex(t *testing.T) {
	t.Setenv("ARCVAULT_CREDENTIAL_KEY", "not-valid-hex!")

	_, err := LoadKey()
	assert.Equal(t, ErrKeyInvalid, err)
}

func TestLoadKey_WrongLength(t *testing.T) {
	// 16 bytes (32 hex chars) instead of 32
	keyHex := "0123456789abcdef0123456789abcdef"
	t.Setenv("ARCVAULT_CREDENTIAL_KEY", keyHex)

	_, err := LoadKey()
	assert.Equal(t, ErrKeyInvalid, err)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	plaintext := []byte("sensitive credential data")

	// Encrypt
	ciphertext, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Decrypt
	decrypted, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	key2 := []byte{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	plaintext := []byte("secret")

	ciphertext, err := Encrypt(key1, plaintext)
	require.NoError(t, err)

	// Try to decrypt with wrong key
	_, err = Decrypt(key2, ciphertext)
	assert.Error(t, err)
}

func TestEncryptDecryptEmpty(t *testing.T) {
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	plaintext := []byte("")

	ciphertext, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	decrypted, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	// GCM decryption returns nil for empty plaintext, which is equivalent
	assert.True(t, len(decrypted) == 0)
}
