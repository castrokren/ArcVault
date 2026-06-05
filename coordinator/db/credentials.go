package db

import (
	"database/sql"
	"fmt"
	"time"
)

type CredentialProfile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	// EncryptedData is NOT exposed in JSON to prevent serialization
	EncryptedData []byte `json:"-"`
}

// CreateCredentialProfile inserts a new credential profile with encrypted data.
func (d *DB) CreateCredentialProfile(id, name, credType string, encryptedData []byte) error {
	_, err := d.conn.Exec(
		`INSERT INTO credential_profiles (id, name, type, encrypted_data) VALUES (?, ?, ?, ?)`,
		id, name, credType, encryptedData,
	)
	if err != nil {
		return fmt.Errorf("failed to create credential profile: %w", err)
	}
	return nil
}

// GetCredentialProfile retrieves a profile by ID (includes encrypted data).
func (d *DB) GetCredentialProfile(id string) (*CredentialProfile, error) {
	var cp CredentialProfile
	err := d.conn.QueryRow(
		`SELECT id, name, type, encrypted_data, created_at FROM credential_profiles WHERE id = ?`,
		id,
	).Scan(&cp.ID, &cp.Name, &cp.Type, &cp.EncryptedData, &cp.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credential profile: %w", err)
	}
	return &cp, nil
}

// ListCredentialProfiles returns all profiles without encrypted data.
func (d *DB) ListCredentialProfiles() ([]*CredentialProfile, error) {
	rows, err := d.conn.Query(
		`SELECT id, name, type, created_at FROM credential_profiles ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list credential profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*CredentialProfile
	for rows.Next() {
		var cp CredentialProfile
		if err := rows.Scan(&cp.ID, &cp.Name, &cp.Type, &cp.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, &cp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profiles: %w", err)
	}

	return profiles, nil
}

// DeleteCredentialProfile removes a profile by ID.
func (d *DB) DeleteCredentialProfile(id string) error {
	result, err := d.conn.Exec(
		`DELETE FROM credential_profiles WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete credential profile: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// HasJobsReferencingProfile checks if any job references the given profile.
func (d *DB) HasJobsReferencingProfile(profileID string) (bool, error) {
	var count int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM jobs WHERE credential_profile_id = ?`,
		profileID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check job references: %w", err)
	}
	return count > 0, nil
}
