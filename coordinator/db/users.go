package db

import (
	"database/sql"
	"time"
)

type User struct {
	ID                   int       `json:"id"`
	Username             string    `json:"username"`
	PasswordHash         string    `json:"-"` // never expose
	Role                 string    `json:"role"`
	MustChangePassword   bool      `json:"must_change_password"`
	CreatedAt            time.Time `json:"created_at"`
}

// CreateUser inserts a new user with the given username, password hash, and role.
func (d *DB) CreateUser(username, passwordHash, role string, mustChange bool) error {
	mustChangeInt := 0
	if mustChange {
		mustChangeInt = 1
	}
	_, err := d.conn.Exec(
		`INSERT INTO users (username, password_hash, role, must_change_password)
		 VALUES (?, ?, ?, ?)`,
		username, passwordHash, role, mustChangeInt,
	)
	return err
}

// GetUserByUsername returns a user by username, or nil if not found.
func (d *DB) GetUserByUsername(username string) (*User, error) {
	var u User
	var mustChangeInt int
	err := d.conn.QueryRow(
		`SELECT id, username, password_hash, role, must_change_password, created_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &mustChangeInt, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.MustChangePassword = mustChangeInt == 1
	return &u, nil
}

// GetUserByID returns a user by ID, or nil if not found.
func (d *DB) GetUserByID(id int) (*User, error) {
	var u User
	var mustChangeInt int
	err := d.conn.QueryRow(
		`SELECT id, username, password_hash, role, must_change_password, created_at
		 FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &mustChangeInt, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.MustChangePassword = mustChangeInt == 1
	return &u, nil
}

// CountUsers returns the total number of users.
func (d *DB) CountUsers() (int, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// UpdatePassword updates a user's password hash and must_change_password flag.
func (d *DB) UpdatePassword(userID int, newHash string, mustChange bool) error {
	mustChangeInt := 0
	if mustChange {
		mustChangeInt = 1
	}
	_, err := d.conn.Exec(
		`UPDATE users SET password_hash=?, must_change_password=? WHERE id=?`,
		newHash, mustChangeInt, userID,
	)
	return err
}

// ListUsers returns all users (excluding password hashes in response).
func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.conn.Query(
		`SELECT id, username, password_hash, role, must_change_password, created_at
		 FROM users ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var mustChangeInt int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &mustChangeInt, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.MustChangePassword = mustChangeInt == 1
		users = append(users, u)
	}
	if users == nil {
		users = []User{}
	}
	return users, rows.Err()
}

// DeleteUser removes a user by ID.
func (d *DB) DeleteUser(userID int) error {
	_, err := d.conn.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}

// UpdateUserRole updates a user's role.
func (d *DB) UpdateUserRole(userID int, role string) error {
	_, err := d.conn.Exec(`UPDATE users SET role=? WHERE id=?`, role, userID)
	return err
}
