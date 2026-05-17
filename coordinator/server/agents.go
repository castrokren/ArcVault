package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type registerRequest struct {
	AgentID  string `json:"agent_id"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
}

type heartbeatRequest struct {
	RollbackAvailable bool `json:"rollback_available"`
}

type agentResponse struct {
	ID               string  `json:"id"`
	Hostname         string  `json:"hostname"`
	OS               string  `json:"os"`
	Arch             string  `json:"arch"`
	Version          string  `json:"version"`
	Status           string  `json:"status"`
	LastSeen         *string `json:"last_seen"`
	RegisteredAt     string  `json:"registered_at"`
	RollbackAvailable bool   `json:"rollback_available"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.AgentID == "" || req.Hostname == "" || req.OS == "" || req.Version == "" {
		http.Error(w, "agent_id, hostname, os, and version are required", http.StatusBadRequest)
		return
	}

	_, err := s.db.Conn().Exec(`
INSERT INTO agents (id, hostname, os, arch, version, status, registered_at)
VALUES (?, ?, ?, ?, ?, 'online', CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
hostname=excluded.hostname,
os=excluded.os,
arch=excluded.arch,
version=excluded.version,
status='online',
last_seen=CURRENT_TIMESTAMP
`, req.AgentID, req.Hostname, req.OS, req.Arch, req.Version)
	if err != nil {
		http.Error(w, "failed to register agent", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "registered",
		"agent_id": req.AgentID,
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	var req heartbeatRequest
	json.NewDecoder(r.Body).Decode(&req)

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.Conn().Exec(`
UPDATE agents SET status='online', last_seen=?, rollback_available=? WHERE id=?
`, now, req.RollbackAvailable, agentID)
	if err != nil {
		http.Error(w, "failed to update heartbeat", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   now,
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	status := q.Get("status")
	p := ParsePagination(r)

	args := []any{}
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

	var total int
	countArgs := append([]any{}, args...)
	if err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM agents"+where, countArgs...).Scan(&total); err != nil {
		http.Error(w, "failed to count agents", http.StatusInternalServerError)
		return
	}

	offset := (p.Page - 1) * p.Limit
	queryArgs := append(args, p.Limit, offset)
	rows, err := s.db.Conn().Query(
		"SELECT id, hostname, os, arch, version, status, last_seen, registered_at, rollback_available FROM agents"+where+
			" ORDER BY registered_at DESC LIMIT ? OFFSET ?",
		queryArgs...,
	)
	if err != nil {
		http.Error(w, "failed to query agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	agents := []agentResponse{}
	for rows.Next() {
		var a agentResponse
		var lastSeen *string
		if err := rows.Scan(&a.ID, &a.Hostname, &a.OS, &a.Arch, &a.Version, &a.Status, &lastSeen, &a.RegisteredAt, &a.RollbackAvailable); err != nil {
			continue
		}
		a.LastSeen = lastSeen
		agents = append(agents, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(agents, total, p.Page, p.Limit))
}
