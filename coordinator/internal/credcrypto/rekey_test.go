package credcrypto

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use a temporary file-based database instead of in-memory
	// to avoid transaction locking issues with modernc.org/sqlite
	tmpFile, err := os.CreateTemp("", "credcrypto-test-*.db")
	require.NoError(t, err)
	tmpFile.Close()

	dbPath := tmpFile.Name()
	t.Cleanup(func() { os.Remove(dbPath) })

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE credentials (
			id TEXT PRIMARY KEY,
			encrypted_value BLOB NOT NULL
		)
	`)
	require.NoError(t, err)

	return db
}

func TestRekeyHappyPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	key1 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	key2 := []byte{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}

	// Insert test credentials encrypted with key1
	plaintext1 := []byte("aws-secret-key")
	plaintext2 := []byte("db-password")

	enc1, err := Encrypt(key1, plaintext1)
	require.NoError(t, err)
	enc2, err := Encrypt(key1, plaintext2)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credentials (id, encrypted_value) VALUES (?, ?)`, "cred1", enc1)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO credentials (id, encrypted_value) VALUES (?, ?)`, "cred2", enc2)
	require.NoError(t, err)

	// Rekey from key1 to key2
	err = Rekey(db, key1, key2)
	require.NoError(t, err)

	// Verify rows are now encrypted with key2
	var retrievedEnc []byte
	err = db.QueryRow(`SELECT encrypted_value FROM credentials WHERE id = ?`, "cred1").Scan(&retrievedEnc)
	require.NoError(t, err)

	// Should decrypt with key2, not key1
	decrypted, err := Decrypt(key2, retrievedEnc)
	require.NoError(t, err)
	assert.Equal(t, plaintext1, decrypted)

	// Verify it no longer decrypts with key1
	_, err = Decrypt(key1, retrievedEnc)
	assert.Error(t, err)

	// Verify second credential was also rekeyed
	err = db.QueryRow(`SELECT encrypted_value FROM credentials WHERE id = ?`, "cred2").Scan(&retrievedEnc)
	require.NoError(t, err)
	decrypted, err = Decrypt(key2, retrievedEnc)
	require.NoError(t, err)
	assert.Equal(t, plaintext2, decrypted)
}

func TestRekeyStopsOnDecryptFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	key1 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	key2 := []byte{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	wrongKey := []byte{99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99, 99}

	// Insert one good credential and one bad one
	plaintext1 := []byte("good-secret")
	enc1, err := Encrypt(key1, plaintext1)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credentials (id, encrypted_value) VALUES (?, ?)`, "good", enc1)
	require.NoError(t, err)

	// Insert a credential encrypted with wrongKey so rekey will fail on it
	badData, err := Encrypt(wrongKey, []byte("bad-secret"))
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO credentials (id, encrypted_value) VALUES (?, ?)`, "bad", badData)
	require.NoError(t, err)

	// Attempt rekey with key1 should fail (can't decrypt the "bad" credential)
	err = Rekey(db, key1, key2)
	assert.Error(t, err)
}

func TestRekeyEmptyTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	key1 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	key2 := []byte{31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}

	// Rekey with no rows should succeed
	err := Rekey(db, key1, key2)
	require.NoError(t, err)
}
