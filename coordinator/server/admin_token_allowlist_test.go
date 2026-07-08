package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The admin token is no longer a master key. After removing the JWTMiddleware
// fallback it must be REJECTED on user/JWT routes (e.g. user management) and
// accepted only on the explicit machine-read allowlist used by the ops scripts.
func TestAdminTokenAllowlist(t *testing.T) {
	s := newTestServer(t) // AdminToken "test-token", JWTSecret "test-secret"

	get := func(path string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rr := httptest.NewRecorder()
		s.router.ServeHTTP(rr, req)
		return rr.Code
	}

	// Rejected: the admin token must not authenticate user management anymore.
	if code := get("/api/users"); code != http.StatusUnauthorized {
		t.Errorf("GET /api/users with admin token: want 401, got %d (master-key fallback still present?)", code)
	}

	// Allowlisted machine reads the deploy/sanity scripts depend on.
	if code := get("/api/agents"); code != http.StatusOK {
		t.Errorf("GET /api/agents with admin token: want 200, got %d", code)
	}
	if code := get("/api/version"); code != http.StatusOK {
		t.Errorf("GET /api/version with admin token: want 200, got %d", code)
	}
}
