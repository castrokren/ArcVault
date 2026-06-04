package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"arcvault/coordinator/business"
)

// === Groups API Endpoints ===

// handleListGroups handles GET /api/groups — viewer+
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groups, err := s.groupService.ListGroups()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list groups"})
		return
	}

	// APIContract: matches dashboard/src/types/api.ts Group interface
	// Last synced: 2026-06-03
	type GroupResponse struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		AgentCount  int    `json:"agent_count"`
	}

	response := make([]GroupResponse, len(groups))
	for i, g := range groups {
		response[i] = GroupResponse{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			AgentCount:  g.AgentCount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleCreateGroup handles POST /api/groups — admin only
// Body: {"name":"...","description":"..."}
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	input := &business.CreateGroupInput{
		Name:        req.Name,
		Description: req.Description,
	}

	group, err := s.groupService.CreateGroup(input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          group.ID,
		"name":        group.Name,
		"description": group.Description,
	})
}

// handleGetGroup handles GET /api/groups/{id} — viewer+
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid group id"})
		return
	}

	group, err := s.groupService.GetGroup(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "group not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          group.ID,
		"name":        group.Name,
		"description": group.Description,
		"agent_count": group.AgentCount,
	})
}

// handleUpdateGroup handles PUT /api/groups/{id} — admin only
// Body: {"name":"...","description":"..."}
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid group id"})
		return
	}

	var req UpdateGroupRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	input := &business.UpdateGroupInput{
		Name:        req.Name,
		Description: req.Description,
	}

	group, err := s.groupService.UpdateGroup(groupID, input)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GroupResponse{
		GroupID:    group.ID,
		Name:       group.Name,
		Description: group.Description,
		AgentCount: group.AgentCount,
		CreatedAt:  group.CreatedAt,
	})
}

// handleDeleteGroup handles DELETE /api/groups/{id} — admin only
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid group id"})
		return
	}

	if err := s.groupService.DeleteGroup(groupID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete group"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleAddAgentToGroup handles POST /api/groups/{id}/agents — admin only
// Body: {"agent_id":"..."}
func (s *Server) handleAddAgentToGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid group id"})
		return
	}

	var req struct {
		AgentID string `json:"agent_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
		return
	}

	if err := s.groupService.AddAgentToGroup(groupID, req.AgentID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"ok": "agent added"})
}

// handleRemoveAgentFromGroup handles DELETE /api/groups/{id}/agents/{agentID} — admin only
func (s *Server) handleRemoveAgentFromGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid group id"})
		return
	}

	agentID := r.PathValue("agentID")

	if err := s.groupService.RemoveAgentFromGroup(groupID, agentID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetGroupAgents handles GET /api/groups/{id}/agents — viewer+
func (s *Server) handleGetGroupAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	groupID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid group id"})
		return
	}

	members, err := s.groupService.GetGroupAgents(groupID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to get group members"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent_ids": members,
	})
}
