package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
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

func (d *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS agents (
	id            TEXT PRIMARY KEY,
	hostname      TEXT NOT NULL,
	os            TEXT NOT NULL,
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
	_, err := d.conn.Exec(schema)
	return err
}
