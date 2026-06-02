package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arcvault/coordinator/config"
	"arcvault/coordinator/db"
)

// newTemplateTestServer creates a test server with an in-memory DB and a
// pre-seeded agent row so template create/update validation passes.
func newTemplateTestServer(t *testing.T) *Server {
	t.Helper()
	database, err := db.Init(":memory:")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	// seed an agent so agent_id validation in handleCreateTemplate passes
	_, err = database.Conn().Exec(
		`INSERT INTO agents (id, hostname, os, arch, version, status, registered_at)
		 VALUES ('agent-01', 'testhost', 'windows', 'amd64', 'v0.5.0', 'online', ?)`,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("failed to seed agent: %v", err)
	}

	cfg := &config.Config{
		Port:       8080,
		AdminToken: "test-token",
	}
	return NewWithFS(cfg, database, nil)
}

// doTemplateRequest is a convenience helper that fires a request through
// adminMiddleware and returns the recorder.
func doTemplateRequest(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("failed to encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)
	return w
}

// seedTemplate creates a template via the API and asserts 201. Returns the response body.
func seedTemplate(t *testing.T, srv *Server, id, name, agentID, command, schedule string) TemplateResponse {
	t.Helper()
	w := doTemplateRequest(t, srv, "POST", "/api/templates", map[string]any{
		"id":       id,
		"name":     name,
		"agent_id": agentID,
		"command":  command,
		"schedule": schedule,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("seedTemplate: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// ── CREATE ────────────────────────────────────────────────────────────────────

func TestCreateTemplate_HappyPath(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "POST", "/api/templates", map[string]any{
		"id":       "nightly-backup",
		"name":     "Nightly Backup",
		"agent_id": "agent-01",
		"command":  "robocopy D:\\Docs E:\\Backup /MIR",
		"schedule": "0 2 * * *",
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID != "nightly-backup" {
		t.Errorf("expected id nightly-backup, got %s", resp.ID)
	}
	if resp.Schedule != "0 2 * * *" {
		t.Errorf("expected schedule '0 2 * * *', got %s", resp.Schedule)
	}
	if !resp.Enabled {
		t.Errorf("expected enabled=true by default")
	}
}

func TestCreateTemplate_DuplicateID(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "dup-id", "First", "agent-01", "cmd /C echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "POST", "/api/templates", map[string]any{
		"id":       "dup-id",
		"name":     "Second",
		"agent_id": "agent-01",
		"command":  "cmd /C echo hi",
		"schedule": "0 1 * * *",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestCreateTemplate_InvalidCron(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "POST", "/api/templates", map[string]any{
		"id":       "bad-cron",
		"name":     "Bad Cron",
		"agent_id": "agent-01",
		"command":  "cmd /C echo hi",
		"schedule": "not-a-cron",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateTemplate_MissingFields(t *testing.T) {
	srv := newTemplateTestServer(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing id", map[string]any{"name": "X", "agent_id": "agent-01", "command": "echo hi", "schedule": "0 1 * * *"}},
		{"missing name", map[string]any{"id": "x", "agent_id": "agent-01", "command": "echo hi", "schedule": "0 1 * * *"}},
		{"missing agent_id", map[string]any{"id": "x", "name": "X", "command": "echo hi", "schedule": "0 1 * * *"}},
		{"missing command", map[string]any{"id": "x", "name": "X", "agent_id": "agent-01", "schedule": "0 1 * * *"}},
		{"missing schedule", map[string]any{"id": "x", "name": "X", "agent_id": "agent-01", "command": "echo hi"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doTemplateRequest(t, srv, "POST", "/api/templates", tc.body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestCreateTemplate_AgentNotFound(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "POST", "/api/templates", map[string]any{
		"id":       "no-agent",
		"name":     "No Agent",
		"agent_id": "ghost-agent",
		"command":  "echo hi",
		"schedule": "0 1 * * *",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── LIST ──────────────────────────────────────────────────────────────────────

func TestListTemplates_Empty(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "GET", "/api/templates", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

func TestListTemplates_Populated(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "t1", "Template One", "agent-01", "echo 1", "0 1 * * *")
	seedTemplate(t, srv, "t2", "Template Two", "agent-01", "echo 2", "0 2 * * *")

	w := doTemplateRequest(t, srv, "GET", "/api/templates", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PaginatedResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
}

func TestListTemplates_SearchByName(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "docs", "Docs Backup", "agent-01", "echo docs", "0 1 * * *")
	seedTemplate(t, srv, "logs", "Logs Archive", "agent-01", "echo logs", "0 2 * * *")

	w := doTemplateRequest(t, srv, "GET", "/api/templates?search=docs", nil)

	var resp PaginatedResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 1 {
		t.Errorf("expected 1 result for search=docs, got %d", resp.Total)
	}
}

func TestListTemplates_SearchNoMatch(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "t1", "Template One", "agent-01", "echo 1", "0 1 * * *")

	w := doTemplateRequest(t, srv, "GET", "/api/templates?search=zzznomatch", nil)

	var resp PaginatedResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 0 {
		t.Errorf("expected 0 results, got %d", resp.Total)
	}
}

func TestListTemplates_Pagination(t *testing.T) {
	srv := newTemplateTestServer(t)
	for i := 0; i < 5; i++ {
		seedTemplate(t, srv,
			"tpl-"+string(rune('a'+i)),
			"Template "+string(rune('A'+i)),
			"agent-01", "echo x", "0 1 * * *",
		)
	}

	w := doTemplateRequest(t, srv, "GET", "/api/templates?page=1&limit=2", nil)

	var resp PaginatedResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
	if resp.Pages != 3 {
		t.Errorf("expected pages=3, got %d", resp.Pages)
	}
}

// ── GET ONE ───────────────────────────────────────────────────────────────────

func TestGetTemplate_Found(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "my-tpl", "My Template", "agent-01", "echo hi", "0 3 * * *")

	w := doTemplateRequest(t, srv, "GET", "/api/templates/my-tpl", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ID != "my-tpl" {
		t.Errorf("expected id=my-tpl, got %s", resp.ID)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "GET", "/api/templates/does-not-exist", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── UPDATE ────────────────────────────────────────────────────────────────────

func TestUpdateTemplate_ChangeSchedule(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "upd-tpl", "Update Me", "agent-01", "echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "PUT", "/api/templates/upd-tpl", map[string]any{
		"schedule": "0 4 * * *",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Schedule != "0 4 * * *" {
		t.Errorf("expected updated schedule '0 4 * * *', got %s", resp.Schedule)
	}
}

func TestUpdateTemplate_Disable(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "dis-tpl", "Disable Me", "agent-01", "echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "PUT", "/api/templates/dis-tpl", map[string]any{
		"enabled": false,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Enabled {
		t.Errorf("expected enabled=false after disable")
	}
}

func TestUpdateTemplate_Reenable(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "reena-tpl", "Re-enable Me", "agent-01", "echo hi", "0 1 * * *")

	// disable first
	doTemplateRequest(t, srv, "PUT", "/api/templates/reena-tpl", map[string]any{"enabled": false})

	// re-enable
	w := doTemplateRequest(t, srv, "PUT", "/api/templates/reena-tpl", map[string]any{"enabled": true})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp TemplateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.Enabled {
		t.Errorf("expected enabled=true after re-enable")
	}
}

func TestUpdateTemplate_InvalidCron(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "cron-tpl", "Cron Test", "agent-01", "echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "PUT", "/api/templates/cron-tpl", map[string]any{
		"schedule": "not-valid-cron",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "PUT", "/api/templates/ghost", map[string]any{
		"name": "Ghost",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── DELETE ────────────────────────────────────────────────────────────────────

func TestDeleteTemplate_Found(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "del-tpl", "Delete Me", "agent-01", "echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "DELETE", "/api/templates/del-tpl", nil)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	// confirm gone
	w2 := doTemplateRequest(t, srv, "GET", "/api/templates/del-tpl", nil)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w2.Code)
	}
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "DELETE", "/api/templates/ghost", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── RUN NOW ───────────────────────────────────────────────────────────────────

func TestRunTemplateNow_Found(t *testing.T) {
	srv := newTemplateTestServer(t)
	seedTemplate(t, srv, "run-tpl", "Run Me", "agent-01", "echo hi", "0 1 * * *")

	w := doTemplateRequest(t, srv, "POST", "/api/templates/run-tpl/run", nil)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}

	// confirm a job row was inserted for agent-01
	var count int
	srv.db.Conn().QueryRow(
		`SELECT COUNT(*) FROM jobs WHERE agent_id = 'agent-01' AND name = 'Run Me'`,
	).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 job row inserted, got %d", count)
	}
}

func TestRunTemplateNow_NotFound(t *testing.T) {
	srv := newTemplateTestServer(t)

	w := doTemplateRequest(t, srv, "POST", "/api/templates/ghost/run", nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
