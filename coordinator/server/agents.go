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
Version  string `json:"version"`
}

type agentResponse struct {
ID           string  `json:"id"`
Hostname     string  `json:"hostname"`
OS           string  `json:"os"`
Version      string  `json:"version"`
Status       string  `json:"status"`
LastSeen     *string `json:"last_seen"`
RegisteredAt string  `json:"registered_at"`
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
INSERT INTO agents (id, hostname, os, version, status, registered_at)
VALUES (?, ?, ?, ?, 'online', CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET
hostname=excluded.hostname,
os=excluded.os,
version=excluded.version,
status='online',
last_seen=CURRENT_TIMESTAMP
`, req.AgentID, req.Hostname, req.OS, req.Version)
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

now := time.Now().UTC().Format(time.RFC3339)
result, err := s.db.Conn().Exec(`
UPDATE agents SET status='online', last_seen=? WHERE id=?
`, now, agentID)
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
rows, err := s.db.Conn().Query(`
SELECT id, hostname, os, version, status, last_seen, registered_at
FROM agents ORDER BY registered_at DESC
`)
if err != nil {
http.Error(w, "failed to query agents", http.StatusInternalServerError)
return
}
defer rows.Close()

agents := []agentResponse{}
for rows.Next() {
var a agentResponse
var lastSeen *string
if err := rows.Scan(&a.ID, &a.Hostname, &a.OS, &a.Version, &a.Status, &lastSeen, &a.RegisteredAt); err != nil {
continue
}
a.LastSeen = lastSeen
agents = append(agents, a)
}

w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(agents)
}