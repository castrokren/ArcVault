package db

import "time"

type AlertRule struct {
	ID        int64     `json:"id"`
	JobID     string    `json:"job_id"`     // empty = applies to all jobs
	RuleType  string    `json:"rule_type"`  // on_failure | duration_exceeded | missed_schedule
	Threshold int       `json:"threshold"`  // seconds; 0 for on_failure
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type AlertHistory struct {
	ID        int64     `json:"id"`
	RuleID    int64     `json:"rule_id"`
	JobID     string    `json:"job_id"`
	RunID     string    `json:"run_id"`
	RuleType  string    `json:"rule_type"`
	FiredAt   time.Time `json:"fired_at"`
	Channel   string    `json:"channel"`
	Status    string    `json:"status"`
	Attempts  int       `json:"attempts"`
	LastError string    `json:"last_error"`
}

func (d *DB) ListAlertRules() ([]AlertRule, error) {
	rows, err := d.conn.Query(`SELECT id, job_id, rule_type, threshold, enabled, created_at FROM alert_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.JobID, &r.RuleType, &r.Threshold, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (d *DB) GetAlertRulesForJob(jobID string) ([]AlertRule, error) {
	// Returns rules for the specific jobID + global rules (where job_id IS NULL)
	rows, err := d.conn.Query(`
		SELECT id, job_id, rule_type, threshold, enabled, created_at
		FROM alert_rules
		WHERE job_id = ? OR job_id IS NULL
		ORDER BY created_at DESC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.JobID, &r.RuleType, &r.Threshold, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (d *DB) CreateAlertRule(r AlertRule) (int64, error) {
	result, err := d.conn.Exec(`
		INSERT INTO alert_rules (job_id, rule_type, threshold, enabled, created_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, r.JobID, r.RuleType, r.Threshold, r.Enabled)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) UpdateAlertRule(r AlertRule) error {
	_, err := d.conn.Exec(`
		UPDATE alert_rules SET job_id = ?, rule_type = ?, threshold = ?, enabled = ? WHERE id = ?
	`, r.JobID, r.RuleType, r.Threshold, r.Enabled, r.ID)
	return err
}

func (d *DB) DeleteAlertRule(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM alert_rules WHERE id = ?`, id)
	return err
}

func (d *DB) AppendAlertHistory(h AlertHistory) (int64, error) {
	result, err := d.conn.Exec(`
		INSERT INTO alert_history (rule_id, job_id, run_id, rule_type, fired_at, channel, status, attempts, last_error)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?)
	`, h.RuleID, h.JobID, h.RunID, h.RuleType, h.Channel, h.Status, h.Attempts, h.LastError)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (d *DB) UpdateAlertHistoryStatus(id int64, status, lastError string, attempts int) error {
	_, err := d.conn.Exec(`
		UPDATE alert_history SET status = ?, last_error = ?, attempts = ? WHERE id = ?
	`, status, lastError, attempts, id)
	return err
}

func (d *DB) ListAlertHistory(limit int) ([]AlertHistory, error) {
	rows, err := d.conn.Query(`
		SELECT id, rule_id, job_id, run_id, rule_type, fired_at, channel, status, attempts, last_error
		FROM alert_history
		ORDER BY fired_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []AlertHistory
	for rows.Next() {
		var h AlertHistory
		if err := rows.Scan(&h.ID, &h.RuleID, &h.JobID, &h.RunID, &h.RuleType, &h.FiredAt, &h.Channel, &h.Status, &h.Attempts, &h.LastError); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (d *DB) PruneAlertHistory(olderThanDays int) error {
	_, err := d.conn.Exec(`
		DELETE FROM alert_history
		WHERE fired_at < datetime('now', ? || ' days')
	`, -olderThanDays)
	return err
}
