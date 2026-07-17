package server

import (
	"encoding/json"
	"log"
	"net/http"

	"arcvault/coordinator/business"
)

// handleCreateAgentToken generates a new token for the given agent.
// Route: POST /api/agents/{id}/token
// Auth: admin-only
func (s *Server) handleCreateAgentToken(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	// Verify agent exists
	_, err := s.db.GetAgent(agentID)
	if err != nil {
		log.Printf("[agent_token] GetAgent failed: %v", err)
		http.Error(w, "agent not found", http.StatusNotFound)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "agent.create_token", ip, false, ptr("agent"), ptr(agentID), ptr("agent not found"))
		return
	}

	// Generate new token
	token, err := s.db.CreateAgentToken(agentID)
	if err != nil {
		log.Printf("[agent_token] CreateAgentToken failed: %v", err)
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "agent.create_token", ip, false, ptr("agent"), ptr(agentID), ptr("token generation failed"))
		return
	}

	// Audit log success
	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "agent.create_token", ip, true, ptr("agent"), ptr(agentID), ptr("generated new token"))

	// Return token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":    token,
		"agent_id": agentID,
	})
}

// ptr is a helper to convert a string to a pointer.
func ptr(s string) *string {
	return &s
}
