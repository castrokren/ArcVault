package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"arcvault/coordinator/db"
)

// handleListAuditCommands returns audit logs with optional filtering.
// Query params:
//   - program: filter by program name (exact match)
//   - whitelisted: "true" | "false" (filter by whitelist status)
//   - agent_id: filter by agent_id
//   - from: RFC3339 timestamp (filter from date)
//   - to: RFC3339 timestamp (filter to date)
//   - limit: results per page (default 100, max 10000)
//   - offset: pagination offset
func (s *Server) handleListAuditCommands(w http.ResponseWriter, r *http.Request) {
	// Build filter from query params
	filter := db.AuditLogFilter{
		ProgramName: r.URL.Query().Get("program"),
		AgentID:     r.URL.Query().Get("agent_id"),
		Limit:       100,
		Offset:      0,
	}

	// Parse whitelist filter if present
	if whitelistedStr := r.URL.Query().Get("whitelisted"); whitelistedStr != "" {
		whitelisted := whitelistedStr == "true"
		filter.IsWhitelisted = &whitelisted
	}

	// Parse time filters
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.FromTime = &t
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.ToTime = &t
		}
	}

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	logs, total, err := s.db.GetAuditLog(filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetNonWhitelistedPrograms returns distinct non-whitelisted programs and their execution counts.
func (s *Server) handleGetNonWhitelistedPrograms(w http.ResponseWriter, r *http.Request) {
	programs, err := s.db.GetNonWhitelistedPrograms()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"programs": programs,
		"count":    len(programs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAuditStats returns audit statistics for a time range.
// Query params:
//   - from: RFC3339 timestamp (default: 24 hours ago)
//   - to: RFC3339 timestamp (default: now)
func (s *Server) handleGetAuditStats(w http.ResponseWriter, r *http.Request) {
	fromTime := time.Now().Add(-24 * time.Hour)
	toTime := time.Now()

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = t
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = t
		}
	}

	stats, err := s.db.GetAuditStats(fromTime, toTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
