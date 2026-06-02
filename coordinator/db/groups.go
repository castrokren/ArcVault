package db

import (
	"database/sql"
	"time"
)

type AgentGroup struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateGroup creates a new agent group.
func (d *DB) CreateGroup(name, description string) (*AgentGroup, error) {
	result, err := d.conn.Exec(
		`INSERT INTO agent_groups (name, description) VALUES (?, ?)`,
		name, description,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return d.GetGroup(int(id))
}

// GetGroup returns a group by ID, or nil if not found.
func (d *DB) GetGroup(id int) (*AgentGroup, error) {
	var g AgentGroup
	err := d.conn.QueryRow(
		`SELECT id, name, description, created_at FROM agent_groups WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// ListGroups returns all agent groups ordered by creation time.
func (d *DB) ListGroups() ([]AgentGroup, error) {
	rows, err := d.conn.Query(
		`SELECT id, name, description, created_at FROM agent_groups ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []AgentGroup
	for rows.Next() {
		var g AgentGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []AgentGroup{}
	}
	return groups, rows.Err()
}

// UpdateGroup updates a group's name and description.
func (d *DB) UpdateGroup(id int, name, description string) error {
	_, err := d.conn.Exec(
		`UPDATE agent_groups SET name=?, description=? WHERE id=?`,
		name, description, id,
	)
	return err
}

// DeleteGroup removes a group by ID (cascades to remove memberships).
func (d *DB) DeleteGroup(id int) error {
	_, err := d.conn.Exec(`DELETE FROM agent_groups WHERE id = ?`, id)
	return err
}

// AddAgentToGroup adds an agent to a group.
func (d *DB) AddAgentToGroup(groupID int, agentID string) error {
	_, err := d.conn.Exec(
		`INSERT INTO agent_group_members (group_id, agent_id) VALUES (?, ?)`,
		groupID, agentID,
	)
	return err
}

// RemoveAgentFromGroup removes an agent from a group.
func (d *DB) RemoveAgentFromGroup(groupID int, agentID string) error {
	_, err := d.conn.Exec(
		`DELETE FROM agent_group_members WHERE group_id = ? AND agent_id = ?`,
		groupID, agentID,
	)
	return err
}

// GetGroupMembers returns all agent IDs in a group.
func (d *DB) GetGroupMembers(groupID int) ([]string, error) {
	rows, err := d.conn.Query(
		`SELECT agent_id FROM agent_group_members WHERE group_id = ?`, groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agentIDs []string
	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agentIDs = append(agentIDs, agentID)
	}
	if agentIDs == nil {
		agentIDs = []string{}
	}
	return agentIDs, rows.Err()
}

// GetAgentGroup returns the group an agent belongs to, or nil if ungrouped.
func (d *DB) GetAgentGroup(agentID string) (*AgentGroup, error) {
	var g AgentGroup
	err := d.conn.QueryRow(
		`SELECT g.id, g.name, g.description, g.created_at
		 FROM agent_groups g
		 INNER JOIN agent_group_members m ON g.id = m.group_id
		 WHERE m.agent_id = ?`, agentID,
	).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}
