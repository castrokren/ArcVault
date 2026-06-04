package db

import (
	"fmt"
	"time"
)

// RegisterAgent inserts or updates an agent registration.
func (d *DB) RegisterAgent(agentID, hostname, os, arch, version, coordinatorID string) error {
	_, err := d.conn.Exec(`
INSERT INTO agents (id, hostname, os, arch, version, status, home_coordinator, registered_at)
VALUES (?, ?, ?, ?, ?, 'online', ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
hostname=excluded.hostname,
os=excluded.os,
arch=excluded.arch,
version=excluded.version,
status='online',
home_coordinator=excluded.home_coordinator,
last_seen=CURRENT_TIMESTAMP
`, agentID, hostname, os, arch, version, coordinatorID)
	return err
}

// UpdateAgentHeartbeat updates agent status, last_seen, and rollback_available.
func (d *DB) UpdateAgentHeartbeat(agentID, coordinatorID string, rollbackAvailable bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := d.conn.Exec(`
UPDATE agents SET status='online', last_seen=?, rollback_available=?, home_coordinator=? WHERE id=?
`, now, rollbackAvailable, coordinatorID, agentID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// GetAgent returns a single agent by ID.
func (d *DB) GetAgent(agentID string) (Agent, error) {
	var a Agent
	err := d.conn.QueryRow(`
SELECT id, hostname, os, arch, version, status, last_seen, registered_at, rollback_available
FROM agents WHERE id = ?
`, agentID).Scan(&a.ID, &a.Hostname, &a.OS, &a.Arch, &a.Version, &a.Status, &a.LastSeen, &a.RegisteredAt, &a.RollbackAvailable)
	if err != nil {
		return Agent{}, err
	}
	return a, nil
}

// ListAgents returns agents with optional search and status filters, with pagination.
func (d *DB) ListAgents(search, status string, limit, offset int) ([]Agent, int, error) {
	args := []interface{}{}
	where := " WHERE 1=1"

	if search != "" {
		where += " AND (id LIKE ? OR hostname LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}

	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}

	// Get total count
	var total int
	countArgs := append([]interface{}{}, args...)
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM agents"+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get paginated results
	queryArgs := append(args, limit, offset)
	rows, err := d.conn.Query(
		"SELECT id, hostname, os, arch, version, status, last_seen, registered_at, rollback_available FROM agents"+where+
			" ORDER BY registered_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	agents := []Agent{}
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Hostname, &a.OS, &a.Arch, &a.Version, &a.Status, &a.LastSeen, &a.RegisteredAt, &a.RollbackAvailable); err != nil {
			continue
		}
		agents = append(agents, a)
	}

	return agents, total, rows.Err()
}

// CountRunningJobs returns the number of jobs with status='running' for the given agent.
func (d *DB) CountRunningJobs(agentID string) (int, error) {
	var count int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM jobs WHERE agent_id = ? AND status = 'running'`, agentID,
	).Scan(&count)
	return count, err
}

// DeleteAgent removes an agent and its tokens + group memberships.
func (d *DB) DeleteAgent(agentID string) error {
	// Delete tokens
	if _, err := d.conn.Exec(`DELETE FROM tokens WHERE agent_id = ?`, agentID); err != nil {
		return err
	}

	// Delete group memberships
	if _, err := d.conn.Exec(`DELETE FROM agent_group_members WHERE agent_id = ?`, agentID); err != nil {
		return err
	}

	// Delete the agent
	_, err := d.conn.Exec(`DELETE FROM agents WHERE id = ?`, agentID)
	return err
}

// DeleteAgentTokens removes all tokens for an agent.
func (d *DB) DeleteAgentTokens(agentID string) error {
	_, err := d.conn.Exec(`DELETE FROM tokens WHERE agent_id = ?`, agentID)
	return err
}

// DeleteAgentGroupMemberships removes all group memberships for an agent.
func (d *DB) DeleteAgentGroupMemberships(agentID string) error {
	_, err := d.conn.Exec(`DELETE FROM agent_group_members WHERE agent_id = ?`, agentID)
	return err
}
