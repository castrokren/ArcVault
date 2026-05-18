package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	AgentID   string    `json:"agent_id"`
	Command   string    `json:"command"`
	Schedule  string    `json:"schedule"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

func Init(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	database := &DB{conn: conn}
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return database, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

// CreateAgentToken generates a new token for the given agent and stores it.
// Multiple tokens per agent are allowed — each call creates a new one.
func (d *DB) CreateAgentToken(agentID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(b)

	_, err := d.conn.Exec(
		`INSERT INTO tokens (token, agent_id, role) VALUES (?, ?, 'agent')`,
		token, agentID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}
	return token, nil
}

// ValidateToken checks if a token exists in the tokens table.
// Returns the role ("agent") if valid, or an error if not found.
func (d *DB) ValidateToken(token string) (string, error) {
	var role string
	err := d.conn.QueryRow(
		`SELECT role FROM tokens WHERE token = ?`, token,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("token not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to validate token: %w", err)
	}
	return role, nil
}

// ListTemplates returns all backup templates ordered by creation time.
func (d *DB) ListTemplates() ([]Template, error) {
	rows, err := d.conn.Query(
		`SELECT id, name, agent_id, command, schedule, enabled, created_at
		 FROM backup_templates ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		var enabled int
		if err := rows.Scan(&t.ID, &t.Name, &t.AgentID, &t.Command, &t.Schedule, &enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []Template{}
	}
	return templates, rows.Err()
}

// GetTemplate returns a single template by ID, or nil if not found.
func (d *DB) GetTemplate(id string) (*Template, error) {
	var t Template
	var enabled int
	err := d.conn.QueryRow(
		`SELECT id, name, agent_id, command, schedule, enabled, created_at
		 FROM backup_templates WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.AgentID, &t.Command, &t.Schedule, &enabled, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	return &t, nil
}

// CreateTemplate inserts a new backup template.
func (d *DB) CreateTemplate(t Template) error {
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := d.conn.Exec(
		`INSERT INTO backup_templates (id, name, agent_id, command, schedule, enabled)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.AgentID, t.Command, t.Schedule, enabled,
	)
	return err
}

// UpdateTemplate updates an existing backup template by ID.
func (d *DB) UpdateTemplate(t Template) error {
	enabled := 0
	if t.Enabled {
		enabled = 1
	}
	_, err := d.conn.Exec(
		`UPDATE backup_templates SET name=?, agent_id=?, command=?, schedule=?, enabled=?
		 WHERE id=?`,
		t.Name, t.AgentID, t.Command, t.Schedule, enabled, t.ID,
	)
	return err
}

// DeleteTemplate removes a backup template by ID.
func (d *DB) DeleteTemplate(id string) error {
	_, err := d.conn.Exec(`DELETE FROM backup_templates WHERE id = ?`, id)
	return err
}

func (d *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS agents (
	id            TEXT PRIMARY KEY,
	hostname      TEXT NOT NULL,
	os            TEXT NOT NULL,
	arch          TEXT NOT NULL DEFAULT '',
	version       TEXT NOT NULL,
	status        TEXT NOT NULL DEFAULT 'offline',
	last_seen     DATETIME,
	registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tokens (
	token      TEXT PRIMARY KEY,
	agent_id   TEXT,
	role       TEXT NOT NULL DEFAULT 'agent',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE TABLE IF NOT EXISTS jobs (
	id          TEXT PRIMARY KEY,
	agent_id    TEXT NOT NULL,
	name        TEXT NOT NULL,
	source_path TEXT NOT NULL,
	dest_path   TEXT NOT NULL,
	schedule    TEXT,
	status      TEXT NOT NULL DEFAULT 'pending',
	created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (agent_id) REFERENCES agents(id)
);

CREATE TABLE IF NOT EXISTS job_runs (
	id          TEXT PRIMARY KEY,
	job_id      TEXT NOT NULL,
	started_at  DATETIME,
	finished_at DATETIME,
	exit_code   INTEGER,
	output      TEXT,
	FOREIGN KEY (job_id) REFERENCES jobs(id)
);
`
	if _, err := d.conn.Exec(schema); err != nil {
		return err
	}
	// Idempotent: add arch column to existing databases that predate this migration.
	d.conn.Exec(`ALTER TABLE agents ADD COLUMN arch TEXT NOT NULL DEFAULT ''`)
	// Idempotent: add rollback_available column for Phase 10 rollback support.
	d.conn.Exec(`ALTER TABLE agents ADD COLUMN rollback_available BOOLEAN NOT NULL DEFAULT 0`)
	// Idempotent: add command column to jobs for Phase 13 template-fired jobs.
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN command TEXT NOT NULL DEFAULT ''`)
	// Idempotent: add backup_templates table for Phase 13 scheduled templates.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS backup_templates (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		agent_id    TEXT NOT NULL,
		command     TEXT NOT NULL,
		schedule    TEXT NOT NULL,
		enabled     INTEGER NOT NULL DEFAULT 1,
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return nil
}
