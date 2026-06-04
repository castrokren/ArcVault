package db

// CreateJob inserts a new job.
func (d *DB) CreateJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string) error {
	_, err := d.conn.Exec(`
INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, jobID, agentID, name, sourcePath, destPath, schedule, syncFlags, status, createdAt)
	return err
}

// CreateGroupJob inserts a new job with group dispatch metadata.
func (d *DB) CreateGroupJob(jobID, agentID, name, sourcePath, destPath string, schedule, syncFlags *string, status, createdAt string, groupID int, dispatchID string) error {
	_, err := d.conn.Exec(`
INSERT INTO jobs (id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at, group_id, dispatch_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, jobID, agentID, name, sourcePath, destPath, schedule, syncFlags, status, createdAt, groupID, dispatchID)
	return err
}

// GetJob returns a single job by ID.
func (d *DB) GetJob(jobID string) (Job, error) {
	var j Job
	err := d.conn.QueryRow(`
SELECT id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at
FROM jobs WHERE id = ?
`, jobID).Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &j.Schedule, &j.SyncFlags, &j.Status, &j.CreatedAt)
	if err != nil {
		return Job{}, err
	}
	return j, nil
}

// ListJobs returns jobs with optional search and status filters, with pagination.
func (d *DB) ListJobs(search, status, agentID string, limit, offset int) ([]Job, int, error) {
	args := []interface{}{}
	where := " WHERE 1=1"

	if search != "" {
		where += " AND (id LIKE ? OR name LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}

	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	if agentID != "" {
		where += " AND agent_id = ?"
		args = append(args, agentID)
	}

	// Get total count
	var total int
	countArgs := append([]interface{}{}, args...)
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM jobs"+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	queryArgs := append(args, limit, offset)
	rows, err := d.conn.Query(
		"SELECT id, agent_id, name, source_path, dest_path, schedule, sync_flags, status, created_at FROM jobs"+where+
			" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	jobs := []Job{}
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.AgentID, &j.Name, &j.SourcePath, &j.DestPath, &j.Schedule, &j.SyncFlags, &j.Status, &j.CreatedAt); err != nil {
			continue
		}
		jobs = append(jobs, j)
	}

	return jobs, total, rows.Err()
}

// UpdateJobStatus updates the status of a job.
func (d *DB) UpdateJobStatus(jobID, status string) error {
	_, err := d.conn.Exec(`UPDATE jobs SET status = ? WHERE id = ?`, status, jobID)
	return err
}

// JobExists checks if a job exists.
func (d *DB) JobExists(jobID string) (bool, error) {
	var exists int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM jobs WHERE id = ?`, jobID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// DeleteJob removes a job by ID.
func (d *DB) DeleteJob(jobID string) error {
	_, err := d.conn.Exec(`DELETE FROM jobs WHERE id = ?`, jobID)
	return err
}

// GetJobName returns the name and agent_id of a job.
func (d *DB) GetJobName(jobID string) (name, agentID string, err error) {
	err = d.conn.QueryRow(`SELECT name, agent_id FROM jobs WHERE id = ?`, jobID).Scan(&name, &agentID)
	return
}

// CreateTemplateJob inserts a transient job fired by a backup template.
// The job carries the template command and is created with status "pending".
func (d *DB) CreateTemplateJob(runID, agentID, name, command, createdAt string) error {
	_, err := d.conn.Exec(
		`INSERT INTO jobs (id, agent_id, name, source_path, dest_path, command, status, created_at)
		 VALUES (?, ?, ?, '', '', ?, 'pending', ?)`,
		runID, agentID, name, command, createdAt,
	)
	return err
}
