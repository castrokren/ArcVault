package server

import (
	"encoding/json"
	"net/http"
	"os"

	"arcvault/coordinator/business"
	"arcvault/coordinator/updater"
)

// handleRollbackAvailable checks if coordinator rollback is available.
func (s *Server) handleRollbackAvailable(w http.ResponseWriter, r *http.Request) {
	available, err := updater.IsRollbackAvailable()
	if err != nil {
		http.Error(w, "failed to check rollback status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"available": available,
	})
}

// handleRollback applies coordinator rollback.
func (s *Server) handleRollback(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	// Check if rollback is available
	available, err := updater.IsRollbackAvailable()
	if err != nil || !available {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "no backup available for rollback",
		})
		s.auditService.LogAction(
			&claims.UserID, claims.Username, claims.Role,
			"coordinator.rollback", ip, false,
			strPtr("coordinator"), nil,
			strPtr("no backup available for rollback"),
		)
		return
	}

	// Get current executable path (same as update flow)
	exePath, err := getCoordinatorBinaryPath()
	if err != nil {
		http.Error(w, "failed to determine coordinator path", http.StatusInternalServerError)
		s.auditService.LogAction(
			&claims.UserID, claims.Username, claims.Role,
			"coordinator.rollback", ip, false,
			strPtr("coordinator"), nil,
			strPtr(err.Error()),
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Emit initial progress event
	w.Write([]byte(`{"type":"rollback_progress","step":"starting","pct":5,"message":"Starting rollback..."}`))
	w.(http.Flusher).Flush()

	// Execute rollback
	progressFn := func(evt updater.ProgressEvent) {
		data, _ := json.Marshal(evt)
		w.Write(data)
		w.Write([]byte("\n"))
		w.(http.Flusher).Flush()
	}

	if err := updater.Rollback(exePath, progressFn); err != nil {
		progressFn(updater.ProgressEvent{
			Type:    "rollback_progress",
			Step:    "error",
			Pct:     100,
			Message: err.Error(),
		})
		s.auditService.LogAction(
			&claims.UserID, claims.Username, claims.Role,
			"coordinator.rollback", ip, false,
			strPtr("coordinator"), nil,
			strPtr(err.Error()),
		)
		return
	}

	s.auditService.LogAction(
		&claims.UserID, claims.Username, claims.Role,
		"coordinator.rollback", ip, true,
		strPtr("coordinator"), nil,
		strPtr("triggered coordinator rollback"),
	)
}

// handleAgentRollback sends rollback command to agent via WebSocket.
func (s *Server) handleAgentRollback(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)
	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "missing agent id", http.StatusBadRequest)
		return
	}

	// Verify agent exists and has rollback available
	var rollbackAvailable bool
	err := s.db.Conn().QueryRow(
		`SELECT rollback_available FROM agents WHERE id=?`,
		agentID,
	).Scan(&rollbackAvailable)

	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		s.auditService.LogAction(
			&claims.UserID, claims.Username, claims.Role,
			"agent.rollback", ip, false,
			strPtr("agent"), &agentID,
			strPtr(err.Error()),
		)
		return
	}

	if !rollbackAvailable {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "no backup available for agent rollback",
		})
		s.auditService.LogAction(
			&claims.UserID, claims.Username, claims.Role,
			"agent.rollback", ip, false,
			strPtr("agent"), &agentID,
			strPtr("no backup available for agent rollback"),
		)
		return
	}

	// Send rollback command through hub
	s.hub.SendToAgent(agentID, map[string]interface{}{
		"type": "rollback_command",
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "rollback_initiated",
		"agent_id": agentID,
	})

	s.auditService.LogAction(
		&claims.UserID, claims.Username, claims.Role,
		"agent.rollback", ip, true,
		strPtr("agent"), &agentID,
		strPtr("triggered agent rollback"),
	)
}

// getCoordinatorBinaryPath returns the path of the coordinator executable.
func getCoordinatorBinaryPath() (string, error) {
	return os.Executable()
}
