package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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

type Federation struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	URL      string     `json:"url"`
	Token    string     `json:"token"`
	Status   string     `json:"status"`
	LastSeen *time.Time `json:"last_seen"`
	Version  string     `json:"version"`
	LastSeq  int64      `json:"last_seq"`
}

func Init(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("could not create db directory: %w", err)
	}

	// WAL mode: concurrent readers + one writer. busy_timeout retries writes for 5s
	// instead of returning SQLITE_BUSY immediately. These two settings together handle
	// all concurrency; no need to strangle the pool with MaxOpenConns(1).
	dsn := dbPath + "?_busy_timeout=5000&_journal_mode=WAL"
	conn, err := sql.Open("sqlite", dsn)
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

	if err := database.EnsureDefaultAdmin(); err != nil {
		return nil, fmt.Errorf("failed to ensure default admin: %w", err)
	}

	return database, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Conn() *sql.DB {
	return d.conn
}

// sqliteTime renders t for storage in a DATETIME column so that it string-compares
// correctly against SQLite's datetime('now'), which is always UTC.
//
// Binding a time.Time directly stores Go's String() form — "2026-07-08 16:24:02
// -0400 EDT" — in local time. That sorts below any UTC datetime('now') string, so
// every `expires_at > datetime('now')` check silently reads false. For revoked
// tokens that fails open: a logged-out token stays valid.
func sqliteTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}

// CreateAgentToken generates a new token for the given agent and stores it.
// Multiple tokens per agent are allowed — each call creates a new one.
// For bootstrap tokens (role starting with "bootstrap"), expires_at is set to 1 hour.
func (d *DB) CreateAgentToken(roleOrAgentID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(b)

	// Check if this is a bootstrap token
	var expiresAt *time.Time
	if strings.HasPrefix(roleOrAgentID, "bootstrap") {
		exp := time.Now().Add(1 * time.Hour)
		expiresAt = &exp
	}

	if expiresAt != nil {
		_, err := d.conn.Exec(
			`INSERT INTO tokens (token, agent_id, role, expires_at) VALUES (?, ?, 'agent', ?)`,
			token, roleOrAgentID, sqliteTime(*expiresAt),
		)
		if err != nil {
			return "", fmt.Errorf("failed to store token: %w", err)
		}
	} else {
		_, err := d.conn.Exec(
			`INSERT INTO tokens (token, agent_id, role) VALUES (?, ?, 'agent')`,
			token, roleOrAgentID,
		)
		if err != nil {
			return "", fmt.Errorf("failed to store token: %w", err)
		}
	}
	return token, nil
}

// ValidateToken checks if a token exists in the tokens table and hasn't expired.
// Returns the role ("agent") if valid, or an error if not found or expired.
func (d *DB) ValidateToken(token string) (string, error) {
	var role string
	err := d.conn.QueryRow(
		`SELECT role FROM tokens WHERE token = ? AND (expires_at IS NULL OR expires_at > datetime('now'))`,
		token,
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("token not found or expired")
	}
	if err != nil {
		return "", fmt.Errorf("failed to validate token: %w", err)
	}
	return role, nil
}

// RevokeToken marks a JWT as revoked by its JTI (JWT ID).
func (d *DB) RevokeToken(jti string, expiresAt time.Time) error {
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO revoked_tokens (jti, expires_at) VALUES (?, ?)`,
		jti, sqliteTime(expiresAt),
	)
	return err
}

// IsTokenRevoked checks if a JWT token (by JTI) has been revoked.
func (d *DB) IsTokenRevoked(jti string) (bool, error) {
	var count int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM revoked_tokens WHERE jti = ? AND expires_at > datetime('now')`,
		jti,
	).Scan(&count)
	return count > 0, err
}

// PruneExpiredTokens removes revoked tokens that have already expired.
func (d *DB) PruneExpiredTokens() error {
	_, err := d.conn.Exec(`DELETE FROM revoked_tokens WHERE expires_at <= datetime('now')`)
	return err
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

// CreateFederation inserts a new federation sub-coordinator record.
func (d *DB) CreateFederation(f Federation) error {
	_, err := d.conn.Exec(
		`INSERT INTO federation (id, name, url, token, status) VALUES (?, ?, ?, ?, 'offline')`,
		f.ID, f.Name, f.URL, f.Token,
	)
	return err
}

// ListFederation returns all registered sub-coordinators.
func (d *DB) ListFederation() ([]Federation, error) {
	rows, err := d.conn.Query(
		`SELECT id, name, url, token, status, last_seen, version, last_seq FROM federation ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Federation
	for rows.Next() {
		var f Federation
		var lastSeen sql.NullTime
		var version sql.NullString
		if err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Token, &f.Status, &lastSeen, &version, &f.LastSeq); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			f.LastSeen = &lastSeen.Time
		}
		if version.Valid {
			f.Version = version.String
		}
		list = append(list, f)
	}
	if list == nil {
		list = []Federation{}
	}
	return list, rows.Err()
}

// GetFederation returns a single federation record by ID, or nil if not found.
func (d *DB) GetFederation(id string) (*Federation, error) {
	var f Federation
	var lastSeen sql.NullTime
	var version sql.NullString
	err := d.conn.QueryRow(
		`SELECT id, name, url, token, status, last_seen, version, last_seq FROM federation WHERE id = ?`, id,
	).Scan(&f.ID, &f.Name, &f.URL, &f.Token, &f.Status, &lastSeen, &version, &f.LastSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastSeen.Valid {
		f.LastSeen = &lastSeen.Time
	}
	if version.Valid {
		f.Version = version.String
	}
	return &f, nil
}

// GetFederationByToken looks up a federation record by its token.
func (d *DB) GetFederationByToken(token string) (*Federation, error) {
	var f Federation
	var lastSeen sql.NullTime
	var version sql.NullString
	err := d.conn.QueryRow(
		`SELECT id, name, url, token, status, last_seen, version, last_seq FROM federation WHERE token = ?`, token,
	).Scan(&f.ID, &f.Name, &f.URL, &f.Token, &f.Status, &lastSeen, &version, &f.LastSeq)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if lastSeen.Valid {
		f.LastSeen = &lastSeen.Time
	}
	if version.Valid {
		f.Version = version.String
	}
	return &f, nil
}

// UpdateFederation updates name, url, and token for an existing federation record.
func (d *DB) UpdateFederation(f Federation) error {
	_, err := d.conn.Exec(
		`UPDATE federation SET name=?, url=?, token=? WHERE id=?`,
		f.Name, f.URL, f.Token, f.ID,
	)
	return err
}

// DeleteFederation removes a federation record by ID.
func (d *DB) DeleteFederation(id string) error {
	_, err := d.conn.Exec(`DELETE FROM federation WHERE id = ?`, id)
	return err
}

// SetFederationStatus updates the live status, last_seen, and version of a sub-coordinator.
func (d *DB) SetFederationStatus(id, status string, lastSeen time.Time, version string) error {
	_, err := d.conn.Exec(
		`UPDATE federation SET status=?, last_seen=?, version=? WHERE id=?`,
		status, lastSeen, version, id,
	)
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

CREATE TABLE IF NOT EXISTS credential_profiles (
	id              TEXT PRIMARY KEY,
	name            TEXT NOT NULL UNIQUE,
	type            TEXT NOT NULL,
	encrypted_data  BLOB NOT NULL,
	created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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
	// Idempotent: add credential_profile_id column to jobs for Path Auth.
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN credential_profile_id TEXT`)
	// Idempotent: add credential profile snapshot columns to job_runs for Path Auth.
	d.conn.Exec(`ALTER TABLE job_runs ADD COLUMN credential_profile_id TEXT`)
	d.conn.Exec(`ALTER TABLE job_runs ADD COLUMN credential_profile_name TEXT`)
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
	// Idempotent: add federation table for Phase 14 multi-coordinator federation.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS federation (
		id        TEXT PRIMARY KEY,
		name      TEXT NOT NULL,
		url       TEXT NOT NULL,
		token     TEXT NOT NULL,
		status    TEXT NOT NULL DEFAULT 'offline',
		last_seen DATETIME,
		version   TEXT
	)`)
	// Idempotent: add users table for Phase 15 RBAC.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS users (
		id                   INTEGER PRIMARY KEY AUTOINCREMENT,
		username             TEXT NOT NULL UNIQUE,
		password_hash        TEXT NOT NULL,
		role                 TEXT NOT NULL CHECK(role IN ('admin','operator','viewer')),
		must_change_password INTEGER NOT NULL DEFAULT 0,
		created_at           DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Idempotent: add agent_groups table for Phase 15 group dispatch.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS agent_groups (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL UNIQUE,
		description TEXT,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Idempotent: add agent_group_members table for Phase 15 group membership.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS agent_group_members (
		group_id  INTEGER NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
		agent_id  TEXT NOT NULL,
		PRIMARY KEY (group_id, agent_id)
	)`)
	// Idempotent: add group_id and dispatch_id columns to jobs for Phase 15 fan-out.
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN group_id    INTEGER REFERENCES agent_groups(id)`)
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN dispatch_id TEXT`)
	// Idempotent: add federation_events table for Phase 16 state sync.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS federation_events (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		seq         INTEGER NOT NULL,
		coordinator TEXT NOT NULL,
		event_type  TEXT NOT NULL,
		payload     TEXT NOT NULL,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Idempotent: add index for federation_events lookups by (coordinator, seq).
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_federation_events_seq ON federation_events(coordinator, seq)`)
	// Idempotent: add last_seq column to federation table for Phase 16 tracking.
	d.conn.Exec(`ALTER TABLE federation ADD COLUMN last_seq INTEGER NOT NULL DEFAULT 0`)
	// Idempotent: add home_coordinator column to agents for Phase 16 health reporting.
	d.conn.Exec(`ALTER TABLE agents ADD COLUMN home_coordinator TEXT NOT NULL DEFAULT ''`)
	// Idempotent: add alert_rules table for Phase 17 alert configuration.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS alert_rules (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id      TEXT,                    -- NULL = applies to all jobs
		rule_type   TEXT NOT NULL,           -- on_failure | duration_exceeded | missed_schedule
		threshold   INTEGER,                 -- seconds (duration_exceeded or missed_schedule)
		enabled     INTEGER NOT NULL DEFAULT 1,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Idempotent: add alert_history table for Phase 17 delivery tracking.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS alert_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_id     INTEGER REFERENCES alert_rules(id),
		job_id      TEXT,
		run_id      TEXT,
		rule_type   TEXT NOT NULL,
		fired_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		channel     TEXT NOT NULL,           -- webhook | email | slack | teams
		status      TEXT NOT NULL,           -- delivered | failed | retrying
		attempts    INTEGER NOT NULL DEFAULT 1,
		last_error  TEXT
	)`)
	// Idempotent: add sync_flags column to jobs for Phase 19 advanced backup options.
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN sync_flags TEXT`)
	// Idempotent: add progress column to jobs for Phase 20 real-time progress tracking.
	d.conn.Exec(`ALTER TABLE jobs ADD COLUMN progress TEXT`)
	// Idempotent: add progress column to job_runs for Phase 21 progress tracking.
	d.conn.Exec(`ALTER TABLE job_runs ADD COLUMN progress INTEGER NOT NULL DEFAULT 0`)
	// Idempotent: add status column to job_runs for Phase 21 progress status tracking.
	d.conn.Exec(`ALTER TABLE job_runs ADD COLUMN status TEXT NOT NULL DEFAULT 'running'`)
	// Idempotent: create job_logs table for Phase 21 backup output logging.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS job_logs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	job_id     TEXT NOT NULL,
	line       TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
)`)
	// Idempotent: create indexes on job_logs for query performance.
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_job_logs_job_id ON job_logs(job_id)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_job_logs_created_at ON job_logs(created_at)`)
	// Idempotent: create trigger to auto-create job_runs when job is inserted (Phase 21).
	d.conn.Exec(`CREATE TRIGGER IF NOT EXISTS auto_job_run AFTER INSERT ON jobs
	 BEGIN
	   INSERT INTO job_runs (id, job_id, started_at) VALUES (
	     'run-' || NEW.id,
	     NEW.id,
	     CURRENT_TIMESTAMP
	   );
	 END`)
	// Idempotent: add revoked_tokens table for JWT revocation on logout.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS revoked_tokens (
		jti TEXT PRIMARY KEY,
		revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires ON revoked_tokens(expires_at)`)
	// Idempotent: add expires_at column to tokens for bootstrap token expiry (short-lived).
	d.conn.Exec(`ALTER TABLE tokens ADD COLUMN expires_at DATETIME`)
	// Idempotent: add command_audit_log table for Phase 2A command audit logging.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS command_audit_log (
		id                  INTEGER PRIMARY KEY AUTOINCREMENT,
		template_id         TEXT,
		job_id              TEXT,
		command_string      TEXT NOT NULL,
		program_name        TEXT NOT NULL,
		is_whitelisted      INTEGER NOT NULL DEFAULT 0,
		mode                TEXT NOT NULL,
		audit_result        TEXT NOT NULL DEFAULT '',
		executed_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
		agent_id            TEXT NOT NULL,
		FOREIGN KEY (job_id) REFERENCES jobs(id),
		FOREIGN KEY (template_id) REFERENCES backup_templates(id)
	)`)
	// Idempotent: add indexes for command_audit_log query performance.
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_command_audit_program ON command_audit_log(program_name)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_command_audit_executed ON command_audit_log(executed_at)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_command_audit_whitelist ON command_audit_log(is_whitelisted)`)
	// Idempotent: add user_audit_log table for user action audit trail.
	d.conn.Exec(`CREATE TABLE IF NOT EXISTS user_audit_log (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id       INTEGER,
		username      TEXT NOT NULL DEFAULT '',
		user_role     TEXT NOT NULL DEFAULT '',
		action        TEXT NOT NULL,
		resource_type TEXT,
		resource_id   TEXT,
		details       TEXT,
		ip_address    TEXT NOT NULL DEFAULT '',
		success       INTEGER NOT NULL DEFAULT 1,
		request_method TEXT,
		request_path  TEXT,
		status_code   INTEGER,
		latency_ms    INTEGER,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_audit_log_action ON user_audit_log(action)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_audit_log_user ON user_audit_log(user_id)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_audit_log_created ON user_audit_log(created_at)`)
	d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_user_audit_log_resource ON user_audit_log(resource_type, resource_id)`)
	return nil
}

// EnsureDefaultAdmin creates a default admin user if no users exist.
// The default credentials are admin/changeme with must_change_password=1.
// This runs on every startup but is a no-op if any users already exist.
func (d *DB) EnsureDefaultAdmin() error {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		// Table may not exist yet in very old DBs — safe to skip
		return nil
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("changeme"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	_, err = d.conn.Exec(
		`INSERT INTO users (username, password_hash, role, must_change_password) VALUES (?, ?, 'admin', 1)`,
		"admin", string(hash),
	)
	if err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	log.Println("Created default admin user (admin/changeme) — please change password on first login")
	return nil
}

// UpdateProgressAndLogs stores job progress percentage, log lines, and status.
// All writes are batched in a single transaction to avoid N round-trips to
// SQLite (which is especially important with SetMaxOpenConns(1)).
func (d *DB) UpdateProgressAndLogs(jobID string, percentage int, logs []string, status string) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`UPDATE job_runs SET progress = ?, status = ? WHERE job_id = ?`,
		percentage, status, jobID,
	); err != nil {
		return err
	}

	if len(logs) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, line := range logs {
			if _, err := stmt.Exec(jobID, line); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// ProgressData holds progress information for a job
type ProgressData struct {
	JobID          string
	Percentage     int
	Status         string
	LastProgressAt *time.Time
	Logs           []string
	LogCount       int
}

// GetProgress retrieves progress data for a job
func (d *DB) GetProgress(jobID string) (*ProgressData, error) {
	var progress int
	var status string
	var lastProgressAt *time.Time

	// Get latest progress from job_runs
	err := d.conn.QueryRow(
		`SELECT COALESCE(progress, 0), COALESCE(status, 'running'), started_at
		 FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT 1`,
		jobID,
	).Scan(&progress, &status, &lastProgressAt)
	if err != nil {
		return nil, err
	}

	// Get recent logs (last 50)
	rows, err := d.conn.Query(
		`SELECT line FROM job_logs WHERE job_id = ? ORDER BY created_at DESC LIMIT 50`,
		jobID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		logs = append(logs, line)
	}

	// Reverse to get chronological order
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	// Count total logs
	var logCount int
	d.conn.QueryRow(
		`SELECT COUNT(*) FROM job_logs WHERE job_id = ?`,
		jobID,
	).Scan(&logCount)

	return &ProgressData{
		JobID:          jobID,
		Percentage:     progress,
		Status:         status,
		LastProgressAt: lastProgressAt,
		Logs:           logs,
		LogCount:       logCount,
	}, nil
}

// JobLogsPage holds paginated log lines for a job
type JobLogsPage struct {
	Logs  []string `json:"logs"`
	Total int      `json:"total"`
}

// GetJobLogsWithPagination retrieves paginated logs for a job
// Returns logs in chronological order (oldest first)
// page is 1-indexed
func (d *DB) GetJobLogsWithPagination(jobID string, page, limit int) (*JobLogsPage, error) {
	// Get total log count
	var total int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM job_logs WHERE job_id = ?`,
		jobID,
	).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Calculate offset for 1-indexed page
	offset := (page - 1) * limit

	// Fetch paginated logs in chronological order
	rows, err := d.conn.Query(
		`SELECT line FROM job_logs WHERE job_id = ? ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		jobID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, err
		}
		logs = append(logs, line)
	}

	// Ensure empty slice instead of nil for JSON marshaling
	if logs == nil {
		logs = []string{}
	}

	return &JobLogsPage{
		Logs:  logs,
		Total: total,
	}, rows.Err()
}
