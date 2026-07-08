package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"arcvault/coordinator/business"
	"arcvault/coordinator/db"
)

// federationResponse merges the DB record with live status from the hub.
type federationResponse struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	URL      string     `json:"url"`
	Status   string     `json:"status"`
	LastSeen *time.Time `json:"last_seen"`
	Version  string     `json:"version"`
}

// federationCacheResponse wraps cached data with stale metadata.
type federationCacheResponse struct {
	Stale bool      `json:"stale"`
	AsOf  time.Time `json:"as_of"`
}

func toFederationResponse(f db.Federation) federationResponse {
	return federationResponse{
		ID:       f.ID,
		Name:     f.Name,
		URL:      f.URL,
		Status:   f.Status,
		LastSeen: f.LastSeen,
		Version:  f.Version,
	}
}

// handleListFederation handles GET /api/federation
func (s *Server) handleListFederation(w http.ResponseWriter, r *http.Request) {
	list, err := s.db.ListFederation()
	if err != nil {
		http.Error(w, "failed to list federation", http.StatusInternalServerError)
		return
	}

	results := make([]federationResponse, len(list))
	for i, f := range list {
		results[i] = toFederationResponse(f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleCreateFederation handles POST /api/federation
func (s *Server) handleCreateFederation(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)

	var input struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, false, strPtr("federation"), nil, strPtr("invalid JSON"))
		return
	}
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, false, strPtr("federation"), nil, strPtr("name is required"))
		return
	}
	if input.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, false, strPtr("federation"), nil, strPtr("url is required"))
		return
	}
	if input.Token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, false, strPtr("federation"), nil, strPtr("token is required"))
		return
	}

	f := db.Federation{
		ID:     newFedID(),
		Name:   input.Name,
		URL:    input.URL,
		Token:  input.Token,
		Status: "offline",
	}

	if err := s.db.CreateFederation(f); err != nil {
		http.Error(w, "failed to create federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, false, strPtr("federation"), nil, strPtr(err.Error()))
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.create", ip, true, strPtr("federation"), &f.ID, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toFederationResponse(f))
}

// handleGetFederation handles GET /api/federation/{id}
func (s *Server) handleGetFederation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f, err := s.db.GetFederation(id)
	if err != nil {
		http.Error(w, "failed to query federation", http.StatusInternalServerError)
		return
	}
	if f == nil {
		http.Error(w, "federation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toFederationResponse(*f))
}

// handleUpdateFederation handles PUT /api/federation/{id}
func (s *Server) handleUpdateFederation(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)
	id := r.PathValue("id")

	existing, err := s.db.GetFederation(id)
	if err != nil {
		http.Error(w, "failed to query federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.update", ip, false, strPtr("federation"), &id, strPtr(err.Error()))
		return
	}
	if existing == nil {
		http.Error(w, "federation not found", http.StatusNotFound)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.update", ip, false, strPtr("federation"), &id, strPtr("federation not found"))
		return
	}

	var input struct {
		Name  *string `json:"name"`
		URL   *string `json:"url"`
		Token *string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.update", ip, false, strPtr("federation"), &id, strPtr("invalid JSON"))
		return
	}

	updated := *existing
	tokenChanged := false
	if input.Name != nil {
		updated.Name = *input.Name
	}
	if input.URL != nil {
		updated.URL = *input.URL
	}
	if input.Token != nil && *input.Token != existing.Token {
		updated.Token = *input.Token
		tokenChanged = true
	}

	if err := s.db.UpdateFederation(updated); err != nil {
		http.Error(w, "failed to update federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.update", ip, false, strPtr("federation"), &id, strPtr(err.Error()))
		return
	}

	// Drop the active connection so the sub reconnects with the new token.
	if tokenChanged {
		s.fedHub.DropConnection(id)
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.update", ip, true, strPtr("federation"), &id, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toFederationResponse(updated))
}

// handleDeleteFederation handles DELETE /api/federation/{id}
func (s *Server) handleDeleteFederation(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)
	id := r.PathValue("id")

	existing, err := s.db.GetFederation(id)
	if err != nil {
		http.Error(w, "failed to query federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.delete", ip, false, strPtr("federation"), &id, strPtr(err.Error()))
		return
	}
	if existing == nil {
		http.Error(w, "federation not found", http.StatusNotFound)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.delete", ip, false, strPtr("federation"), &id, strPtr("federation not found"))
		return
	}

	s.fedHub.DropConnection(id)

	if err := s.db.DeleteFederation(id); err != nil {
		http.Error(w, "failed to delete federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.delete", ip, false, strPtr("federation"), &id, strPtr(err.Error()))
		return
	}

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.delete", ip, true, strPtr("federation"), &id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// handleSyncFederation handles POST /api/federation/{id}/sync
// Drops the active connection — the sub reconnects and sends a fresh snapshot.
func (s *Server) handleSyncFederation(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	ip := business.ClientIP(r)
	id := r.PathValue("id")

	f, err := s.db.GetFederation(id)
	if err != nil {
		http.Error(w, "failed to query federation", http.StatusInternalServerError)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.sync", ip, false, strPtr("federation"), &id, strPtr(err.Error()))
		return
	}
	if f == nil {
		http.Error(w, "federation not found", http.StatusNotFound)
		s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.sync", ip, false, strPtr("federation"), &id, strPtr("federation not found"))
		return
	}

	s.fedHub.DropConnection(id)

	s.auditService.LogAction(&claims.UserID, claims.Username, claims.Role, "federation.sync", ip, true, strPtr("federation"), &id, nil)

	w.WriteHeader(http.StatusAccepted)
}

// handleFederationAgents handles GET /api/federation/{id}/agents
func (s *Server) handleFederationAgents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.requireFederationExists(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cache, ok := s.fedHub.GetCache(id)
	if !ok {
		// Sub is offline — try to serve from a disconnected cache stored in hub's offline map.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"agents": []agentResponse{},
			"stale":  true,
			"as_of":  time.Now(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"agents": cache.Agents,
		"stale":  cache.Stale,
		"as_of":  cache.AsOf,
	})
}

// handleFederationJobs handles GET /api/federation/{id}/jobs
func (s *Server) handleFederationJobs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.requireFederationExists(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cache, ok := s.fedHub.GetCache(id)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"jobs":  []Job{},
			"stale": true,
			"as_of": time.Now(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jobs":  cache.Jobs,
		"stale": cache.Stale,
		"as_of": cache.AsOf,
	})
}

// handleFederationHistory handles GET /api/federation/{id}/history
func (s *Server) handleFederationHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.requireFederationExists(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	cache, ok := s.fedHub.GetCache(id)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"history": []JobRun{},
			"stale":   true,
			"as_of":   time.Now(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"history": cache.History,
		"stale":   cache.Stale,
		"as_of":   cache.AsOf,
	})
}

// requireFederationExists returns an error if the federation record doesn't exist.
func (s *Server) requireFederationExists(id string) error {
	f, err := s.db.GetFederation(id)
	if err != nil {
		return fmt.Errorf("failed to query federation")
	}
	if f == nil {
		return fmt.Errorf("federation not found")
	}
	return nil
}

// newFedID generates a simple unique ID for a federation record.
func newFedID() string {
	return fmt.Sprintf("fed-%d", time.Now().UnixNano())
}
