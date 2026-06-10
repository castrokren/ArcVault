package credcrypto

import (
	"database/sql"
	"fmt"
)

// Rekey rotates the encryption key for all encrypted credentials.
// Reads all rows from the credentials table, decrypts with oldKey, encrypts with newKey.
// Updates are applied row-by-row; on any decryption error, the operation stops without completing all rows.
func Rekey(db *sql.DB, oldKey, newKey []byte) error {
	rows, err := db.Query(`SELECT id, encrypted_value FROM credentials`)
	if err != nil {
		return fmt.Errorf("failed to query credentials: %w", err)
	}

	var updates []struct {
		id    string
		value []byte
	}

	// Decrypt all rows and prepare updates
	for rows.Next() {
		var id string
		var encryptedValue []byte
		if err := rows.Scan(&id, &encryptedValue); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Decrypt with old key
		plaintext, err := Decrypt(oldKey, encryptedValue)
		if err != nil {
			rows.Close()
			return fmt.Errorf("failed to decrypt credential %s: %w", id, err)
		}

		// Encrypt with new key
		newEncrypted, err := Encrypt(newKey, plaintext)
		if err != nil {
			rows.Close()
			return fmt.Errorf("failed to encrypt credential %s: %w", id, err)
		}

		updates = append(updates, struct {
			id    string
			value []byte
		}{id: id, value: newEncrypted})
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("error iterating rows: %w", err)
	}
	rows.Close()

	// Apply all updates
	for _, update := range updates {
		_, err := db.Exec(`UPDATE credentials SET encrypted_value = ? WHERE id = ?`, update.value, update.id)
		if err != nil {
			return fmt.Errorf("failed to update credential %s: %w", update.id, err)
		}
	}

	return nil
}
