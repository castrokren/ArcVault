package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"arcvault/coordinator/db"

	"github.com/robfig/cron/v3"
)

// TemplateResponse is the API shape for a backup template, including the
// computed next_run field derived from the live cron entry.
type TemplateResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	AgentID   string     `json:"agent_id"`
	Command   string     `json:"command"`
	Schedule  string     `json:"schedule"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	NextRun   *time.Time `json:"next_run"`
}

func (s *Server) templateToResponse(t db.Template) TemplateResponse {
	return TemplateResponse{
		ID:        t.ID,
		Name:      t.Name,
		AgentID:   t.AgentID,
		Command:   t.Command,
		Schedule:  t.Schedule,
		Enabled:   t.Enabled,
		CreatedAt: t.CreatedAt,
		NextRun:   s.NextTemplateRun(t.ID),
	}
}

// handleListTemplates handles GET /api/templates
// Optional query params: ?search=, ?page=, ?limit=
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	search := strings.ToLower(r.URL.Query().Get("search"))
	p := ParsePagination(r)

	templates, err := s.db.ListTemplates()
	if err != nil {
		http.Error(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	// filter by search
	filtered := []db.Template{}
	for _, t := range templates {
		if search == "" ||
			strings.Contains(strings.ToLower(t.Name), search) ||
			strings.Contains(strings.ToLower(t.AgentID), search) {
			filtered = append(filtered, t)
		}
	}

	total := len(filtered)
	start := (p.Page - 1) * p.Limit
	end := start + p.Limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	page := filtered[start:end]

	results := make([]TemplateResponse, len(page))
	for i, t := range page {
		results[i] = s.templateToResponse(t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NewPaginatedResponse(results, total, p.Page, p.Limit))
}

// handleCreateTemplate handles POST /api/templates
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		AgentID  string `json:"agent_id"`
		Command  string `json:"command"`
		Schedule string `json:"schedule"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// validation
	if input.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.AgentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}
	if input.Command == "" {
		http.Error(w, "command is required", http.StatusBadRequest)
		return
	}
	if input.Schedule == "" {
		http.Error(w, "schedule is required", http.StatusBadRequest)
		return
	}
	if _, err := cron.ParseStandard(input.Schedule); err != nil {
		http.Error(w, "invalid cron expression: "+err.Error(), http.StatusBadRequest)
		return
	}

	// verify agent exists
	var agentExists bool
	s.db.Conn().QueryRow(`SELECT COUNT(*) > 0 FROM agents WHERE id = ?`, input.AgentID).Scan(&agentExists)
	if !agentExists {
		http.Error(w, "agent not found", http.StatusBadRequest)
		return
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	t := db.Template{
		ID:       input.ID,
		Name:     input.Name,
		AgentID:  input.AgentID,
		Command:  input.Command,
		Schedule: input.Schedule,
		Enabled:  enabled,
	}

	if err := s.db.CreateTemplate(t); err != nil {
		// SQLite UNIQUE constraint violation
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "template id already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create template", http.StatusInternalServerError)
		return
	}

	if err := s.AddTemplateSchedule(t); err != nil {
		// non-fatal — template is saved, scheduler may not be started yet in tests
	}

	// re-fetch to get created_at from DB
	created, err := s.db.GetTemplate(t.ID)
	if err != nil || created == nil {
		http.Error(w, "failed to fetch created template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s.templateToResponse(*created))
}

// handleGetTemplate handles GET /api/templates/{id}
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.db.GetTemplate(id)
	if err != nil {
		http.Error(w, "failed to query template", http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.templateToResponse(*t))
}

// handleUpdateTemplate handles PUT /api/templates/{id}
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.db.GetTemplate(id)
	if err != nil {
		http.Error(w, "failed to query template", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	var input struct {
		Name     *string `json:"name"`
		AgentID  *string `json:"agent_id"`
		Command  *string `json:"command"`
		Schedule *string `json:"schedule"`
		Enabled  *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// apply partial updates
	updated := *existing
	if input.Name != nil {
		updated.Name = *input.Name
	}
	if input.AgentID != nil {
		updated.AgentID = *input.AgentID
	}
	if input.Command != nil {
		updated.Command = *input.Command
	}
	if input.Schedule != nil {
		if _, err := cron.ParseStandard(*input.Schedule); err != nil {
			http.Error(w, "invalid cron expression: "+err.Error(), http.StatusBadRequest)
			return
		}
		updated.Schedule = *input.Schedule
	}
	if input.Enabled != nil {
		updated.Enabled = *input.Enabled
	}

	if err := s.db.UpdateTemplate(updated); err != nil {
		http.Error(w, "failed to update template", http.StatusInternalServerError)
		return
	}

	if err := s.AddTemplateSchedule(updated); err != nil {
		// non-fatal
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.templateToResponse(updated))
}

// handleDeleteTemplate handles DELETE /api/templates/{id}
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := s.db.GetTemplate(id)
	if err != nil {
		http.Error(w, "failed to query template", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	s.RemoveTemplateSchedule(id)

	if err := s.db.DeleteTemplate(id); err != nil {
		http.Error(w, "failed to delete template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRunTemplateNow handles POST /api/templates/{id}/run
func (s *Server) handleRunTemplateNow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	t, err := s.db.GetTemplate(id)
	if err != nil {
		http.Error(w, "failed to query template", http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	s.fireTemplate(*t)

	w.WriteHeader(http.StatusAccepted)
}
