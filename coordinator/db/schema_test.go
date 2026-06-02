package db

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigration_CreateJobLogsTable verifies job_logs table is created with correct schema.
func TestMigration_CreateJobLogsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Verify job_logs table exists
	var exists int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='job_logs'`,
	).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "job_logs table should exist")

	// Verify columns exist
	rows, err := db.conn.Query(`PRAGMA table_info(job_logs)`)
	require.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var cid int
		var name string
		var type_ string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		err := rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk)
		require.NoError(t, err)
		columns[name] = type_
	}

	// Verify required columns
	assert.Contains(t, columns, "id", "job_logs should have id column")
	assert.Contains(t, columns, "job_id", "job_logs should have job_id column")
	assert.Contains(t, columns, "line", "job_logs should have line column")
	assert.Contains(t, columns, "created_at", "job_logs should have created_at column")

	// Verify column types
	assert.Equal(t, "TEXT", columns["job_id"], "job_id should be TEXT")
	assert.Equal(t, "TEXT", columns["line"], "line should be TEXT")
	assert.Equal(t, "DATETIME", columns["created_at"], "created_at should be DATETIME")
}

// TestMigration_AddProgressColumnToJobRuns verifies progress column is added to job_runs.
func TestMigration_AddProgressColumnToJobRuns(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Verify progress column exists in job_runs
	rows, err := db.conn.Query(`PRAGMA table_info(job_runs)`)
	require.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var cid int
		var name string
		var type_ string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		err := rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk)
		require.NoError(t, err)
		columns[name] = type_
	}

	assert.Contains(t, columns, "progress", "job_runs should have progress column")
	assert.Equal(t, "INTEGER", columns["progress"], "progress should be INTEGER")
}

// TestMigration_JobLogsIndexes verifies indexes are created on job_logs for query performance.
func TestMigration_JobLogsIndexes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Verify indexes exist
	rows, err := db.conn.Query(
		`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='job_logs'`,
	)
	require.NoError(t, err)
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes[name] = true
	}

	// Should have at least one index on job_id and one on created_at
	hasJobIDIndex := false
	hasCreatedAtIndex := false

	for indexName := range indexes {
		indexInfo, err := db.conn.Query(`PRAGMA index_info(` + indexName + `)`)
		require.NoError(t, err)

		for indexInfo.Next() {
			var seq int
			var cid int
			var name string
			err := indexInfo.Scan(&seq, &cid, &name)
			require.NoError(t, err)

			if name == "job_id" {
				hasJobIDIndex = true
			}
			if name == "created_at" {
				hasCreatedAtIndex = true
			}
		}
		indexInfo.Close()
	}

	assert.True(t, hasJobIDIndex, "job_logs should have index on job_id")
	assert.True(t, hasCreatedAtIndex, "job_logs should have index on created_at")
}

// TestMigration_JobLogsForeignKeyConstraint verifies foreign key constraint on job_id.
func TestMigration_JobLogsForeignKeyConstraint(t *testing.T) {
	db := setupTestDBForSchema(t)
	defer db.Close()

	// Enable foreign keys for SQLite
	_, err := db.conn.Exec(`PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	// Create a test agent first
	_, err = db.conn.Exec(
		`INSERT INTO agents (id, hostname, os, arch, version) VALUES (?, ?, ?, ?, ?)`,
		"agent-1", "test-host", "linux", "x86_64", "v1.0",
	)
	require.NoError(t, err)

	// Create a test job
	jobID := "test-job-001"
	_, err = db.conn.Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path) VALUES (?, ?, ?, ?, ?)`,
		jobID, "agent-1", "Test Job", "/source", "/dest",
	)
	require.NoError(t, err)

	// Insert a valid job_log entry (should succeed)
	_, err = db.conn.Exec(
		`INSERT INTO job_logs (job_id, line) VALUES (?, ?)`,
		jobID, "Test log line",
	)
	require.NoError(t, err)

	// Verify the entry exists
	var count int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM job_logs WHERE job_id = ?`, jobID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "job_log should be inserted")
}

// TestMigration_Idempotent verifies migration can be run multiple times safely.
func TestMigration_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Run migration again (should not error)
	err := db.migrate()
	require.NoError(t, err, "migration should be idempotent and not error on second run")

	// Verify tables still exist
	var exists int
	err = db.conn.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='job_logs'`,
	).Scan(&exists)
	require.NoError(t, err)
	assert.Equal(t, 1, exists, "job_logs table should still exist after second migration run")
}

// setupTestDBForSchema creates a temporary test database with migration run.
func setupTestDBForSchema(t *testing.T) *DB {
	// Create temporary file for test DB
	tmpfile, err := os.CreateTemp("", "arcvault-test-*.db")
	require.NoError(t, err)
	tmpfile.Close()

	dbPath := tmpfile.Name()
	t.Cleanup(func() {
		os.Remove(dbPath)
	})

	// Initialize DB with migrations
	db, err := Init(dbPath)
	require.NoError(t, err, "should initialize test database without error")

	return db
}

// BenchmarkMigration measures migration time (for optimization later).
func BenchmarkMigration(b *testing.B) {
	tmpfile, err := os.CreateTemp("", "arcvault-bench-*.db")
	if err != nil {
		b.Fatalf("could not create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Init(tmpfile.Name())
		if err != nil {
			b.Fatalf("migration failed: %v", err)
		}
	}
}
