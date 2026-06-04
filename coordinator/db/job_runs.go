package db

import (
	"database/sql"
)

// ListJobRuns returns job runs for a specific job with pagination.
func (d *DB) ListJobRuns(jobID string, limit, offset int) ([]JobRun, int, error) {
	// Get total count
	var total int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM job_runs WHERE job_id = ?`,
		jobID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	rows, err := d.conn.Query(
		`SELECT id, job_id, started_at, finished_at, status, exit_code, output FROM job_runs WHERE job_id = ? ORDER BY started_at DESC LIMIT ? OFFSET ?`,
		jobID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var run JobRun
		if err := rows.Scan(&run.ID, &run.JobID, &run.StartedAt, &run.FinishedAt, &run.Status, &run.ExitCode, &run.Output); err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}

	return runs, total, rows.Err()
}

// GetFirstJobRun returns the ID of the first job run for a job (created by trigger),
// or empty string if no run exists.
func (d *DB) GetFirstJobRun(jobID string) (string, error) {
	var runID string
	err := d.conn.QueryRow(
		`SELECT id FROM job_runs WHERE job_id = ? ORDER BY started_at ASC LIMIT 1`,
		jobID,
	).Scan(&runID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return runID, err
}

// CreateJobRun inserts a new job run with initial result data.
func (d *DB) CreateJobRun(id, jobID string, exitCode int, output, startedAt, finishedAt string) error {
	_, err := d.conn.Exec(
		`INSERT INTO job_runs (id, job_id, exit_code, output, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, jobID, exitCode, output, startedAt, finishedAt,
	)
	return err
}

// UpdateJobRun updates an existing job run with result data.
func (d *DB) UpdateJobRun(id string, exitCode int, output, startedAt, finishedAt string) error {
	_, err := d.conn.Exec(
		`UPDATE job_runs SET exit_code = ?, output = ?, started_at = ?, finished_at = ? WHERE id = ?`,
		exitCode, output, startedAt, finishedAt, id,
	)
	return err
}

// CountJobRuns returns the total number of runs for a job.
func (d *DB) CountJobRuns(jobID string) (int, error) {
	var count int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM job_runs WHERE job_id = ?`,
		jobID,
	).Scan(&count)
	return count, err
}

// ListAllJobRuns returns job runs with filters and pagination (for Reports).
// Supported filter keys: "job_id", "agent_id", "status".
func (d *DB) ListAllJobRuns(filters map[string]string, limit, offset int) ([]JobRun, int, error) {
	// Build dynamic query based on filters
	join := ""
	where := " WHERE 1=1"
	var args []interface{}

	if agentID, ok := filters["agent_id"]; ok && agentID != "" {
		join = " JOIN jobs j ON job_runs.job_id = j.id"
		where += " AND j.agent_id = ?"
		args = append(args, agentID)
	}
	if jid, ok := filters["job_id"]; ok && jid != "" {
		where += " AND job_runs.job_id = ?"
		args = append(args, jid)
	}
	if st, ok := filters["status"]; ok && st != "" {
		where += " AND job_runs.status = ?"
		args = append(args, st)
	}

	base := `SELECT job_runs.id, job_runs.job_id, job_runs.started_at, job_runs.finished_at, job_runs.status, job_runs.exit_code, job_runs.output FROM job_runs` + join + where

	// Count
	countArgs := append([]interface{}{}, args...)
	var total int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM job_runs"+join+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add sorting and pagination to main query
	query := base + ` ORDER BY job_runs.started_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []JobRun
	for rows.Next() {
		var run JobRun
		if err := rows.Scan(&run.ID, &run.JobID, &run.StartedAt, &run.FinishedAt, &run.Status, &run.ExitCode, &run.Output); err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}

	return runs, total, rows.Err()
}
