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

// APIContract: matches dashboard/src/types/api.ts Agent interface
// Last synced: 2026-06-03
type agentResponse struct {
	ID                string  `json:"id"`
	Hostname          string  `json:"hostname"`
	OS                string  `json:"os"`
	Arch              string  `json:"arch"`
	Version           string  `json:"version"`
	Status            string  `json:"status"`
	LastSeen          *string `json:"last_seen"`
	RegisteredAt      string  `json:"registered_at"`
	RollbackAvailable bool    `json:"rollback_available"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		s.logAudit(r, claims, "agent.register", false, strPtr("agent"), nil)
		return
	}

	agent, err := s.agentService.RegisterAgent(req.AgentID, req.Hostname, req.OS, req.Arch, req.Version, s.coordinatorID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.logAudit(r, claims, "agent.register", false, strPtr("agent"), strPtr(req.AgentID))
		return
	}

	// Broadcast to root coordinator if running as a sub.
	payload, _ := json.Marshal(FedAgentRegistered{
		Agent: agentResponse{
			ID:                agent.ID,
			Hostname:          agent.Hostname,
			OS:                agent.OS,
			Arch:              agent.Arch,
			Version:           agent.Version,
			Status:            agent.Status,
			RollbackAvailable: agent.RollbackAvailable,
		},
	})
	s.broadcastFedDelta(FedMessage{
		Type:    FedEventAgentRegistered,
		Payload: json.RawMessage(payload),
	})

	// Append to federation_events log for state sync.
	s.db.AppendFederationEvent(s.coordinatorID, "agent_registered", string(payload))

	s.logAudit(r, claims, "agent.register", true, strPtr("agent"), strPtr(req.AgentID))

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

	if err := s.agentService.UpdateAgentHeartbeat(agentID, s.coordinatorID, req.RollbackAvailable); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Broadcast heartbeat delta to root coordinator if running as a sub.
	hbPayload, _ := json.Marshal(FedAgentHeartbeat{
		AgentID:  agentID,
		Status:   "online",
		LastSeen: &now,
	})
	s.broadcastFedDelta(FedMessage{
		Type:    FedEventAgentHeartbeat,
		Payload: json.RawMessage(hbPayload),
	})

	// Append to federation_events log for state sync.
	s.db.AppendFederationEvent(s.coordinatorID, "agent_heartbeat", string(hbPayload))

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

	offset := (p.Page - 1) * p.Limit
	result, err := s.agentService.ListAgents(search, status, p.Limit, offset)
	if err != nil {
		http.Error(w, "failed to list agents", http.StatusInternalServerError)
		return
	}

	// Convert service DTOs to response DTOs
	agents := make([]agentResponse, len(result.Agents))
	for i, a := range result.Agents {
		agents[i] = agentResponse{
			ID:                a.ID,
			Hostname:          a.Hostname,
			OS:                a.OS,
			Arch:              a.Arch,
			Version:           a.Version,
			Status:            a.Status,
			LastSeen:          a.LastSeen,
			RegisteredAt:      a.RegisteredAt,
			RollbackAvailable: a.RollbackAvailable,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(agents, result.Total, result.Page, result.Limit))
}

// handleDeleteAgent handles DELETE /api/agents/{id} — admin only.
// Blocks deletion if the agent has any currently running jobs.
// Cleans up tokens and group memberships on success.
// Historical jobs are left intact (they retain run history).
func (s *Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	agentID := r.PathValue("id")
	if agentID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing agent id"})
		s.logAudit(r, claims, "agent.delete", false, strPtr("agent"), nil)
		return
	}

	err := s.agentService.DeleteAgent(agentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		// Determine HTTP status code based on error message
		statusCode := http.StatusInternalServerError
		if err.Error() == "agent not found" {
			statusCode = http.StatusNotFound
		} else if len(err.Error()) > len("agent has ") && err.Error()[:len("agent has ")] == "agent has " {
			statusCode = http.StatusConflict
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		s.logAudit(r, claims, "agent.delete", false, strPtr("agent"), strPtr(agentID))
		return
	}

	s.logAudit(r, claims, "agent.delete", true, strPtr("agent"), strPtr(agentID))

	w.WriteHeader(http.StatusNoContent)
}
