package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"arcvault/coordinator/business"
	"arcvault/coordinator/db"
)

// handleListAlertRules lists all alert rules (viewer+)
func (s *Server) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.db.ListAlertRules()
	if err != nil {
		log.Printf("[alert_rules] list failed: %v", err)
		http.Error(w, "failed to list rules", http.StatusInternalServerError)
		return
	}

	if rules == nil {
		rules = []db.AlertRule{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

// handleCreateAlertRule creates a new alert rule (admin)
func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	var input struct {
		JobID     string `json:"job_id"`
		RuleType  string `json:"rule_type"`
		Threshold int    `json:"threshold"`
		Enabled   bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.create", ip, false, strPtr("alert_rule"), nil, strPtr("invalid JSON"))
		return
	}

	// Validate rule_type
	validTypes := map[string]bool{
		"on_failure":        true,
		"duration_exceeded": true,
		"missed_schedule":   true,
	}
	if !validTypes[input.RuleType] {
		http.Error(w, "invalid rule_type", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.create", ip, false, strPtr("alert_rule"), nil, strPtr("invalid rule_type"))
		return
	}

	rule := db.AlertRule{
		JobID:     input.JobID,
		RuleType:  input.RuleType,
		Threshold: input.Threshold,
		Enabled:   input.Enabled,
	}

	id, err := s.db.CreateAlertRule(rule)
	if err != nil {
		log.Printf("[alert_rules] create failed: %v", err)
		http.Error(w, "failed to create rule", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.create", ip, false, strPtr("alert_rule"), nil, strPtr(err.Error()))
		return
	}

	rule.ID = id
	idStr := strconv.FormatInt(id, 10)
	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.create", ip, true, strPtr("alert_rule"), &idStr, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

// handleUpdateAlertRule updates an alert rule (admin)
func (s *Server) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.update", ip, false, strPtr("alert_rule"), nil, strPtr("invalid id"))
		return
	}

	var input struct {
		JobID     string `json:"job_id"`
		RuleType  string `json:"rule_type"`
		Threshold int    `json:"threshold"`
		Enabled   bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.update", ip, false, strPtr("alert_rule"), &idStr, strPtr("invalid JSON"))
		return
	}

	rule := db.AlertRule{
		ID:        id,
		JobID:     input.JobID,
		RuleType:  input.RuleType,
		Threshold: input.Threshold,
		Enabled:   input.Enabled,
	}

	if err := s.db.UpdateAlertRule(rule); err != nil {
		log.Printf("[alert_rules] update failed: %v", err)
		http.Error(w, "failed to update rule", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.update", ip, false, strPtr("alert_rule"), &idStr, strPtr(err.Error()))
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.update", ip, true, strPtr("alert_rule"), &idStr, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

// handleDeleteAlertRule deletes an alert rule (admin)
func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.delete", ip, false, strPtr("alert_rule"), nil, strPtr("invalid id"))
		return
	}

	if err := s.db.DeleteAlertRule(id); err != nil {
		log.Printf("[alert_rules] delete failed: %v", err)
		http.Error(w, "failed to delete rule", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.delete", ip, false, strPtr("alert_rule"), &idStr, strPtr(err.Error()))
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "alert_rule.delete", ip, true, strPtr("alert_rule"), &idStr, nil)

	w.WriteHeader(http.StatusNoContent)
}
