package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"arcvault/coordinator/config"
)

// attemptLogin fires one login at handleLogin from the given source IP.
func attemptLogin(s *Server, username, srcIP string) int {
	body := fmt.Sprintf(`{"username":%q,"password":"wrongpassword123"}`, username)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.RemoteAddr = srcIP + ":40000"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleLogin(rr, req)
	return rr.Code
}

// TestPerAccountThrottle_survivesIPRotation is the point of the account key:
// every request comes from a fresh IP, so the per-IP limiter never trips, but
// the account limiter must still cut the attack off.
func TestPerAccountThrottle_survivesIPRotation(t *testing.T) {
	s := newTestServer(t)

	// Burst budget is 5; the 6th attempt on the same account must be throttled
	// even though this IP has never been seen before.
	for i := 1; i <= loginBurst; i++ {
		ip := fmt.Sprintf("198.51.100.%d", i)
		if got := attemptLogin(s, "admin", ip); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d from %s: got %d, want 401", i, ip, got)
		}
	}

	freshIP := "198.51.100.99"
	if got := attemptLogin(s, "admin", freshIP); got != http.StatusTooManyRequests {
		t.Fatalf("attempt %d from fresh IP %s: got %d, want 429 — "+
			"a distributed attack on one account is unthrottled", loginBurst+1, freshIP, got)
	}
}

// A throttled account must not throttle unrelated accounts.
func TestPerAccountThrottle_isolatesAccounts(t *testing.T) {
	s := newTestServer(t)

	for i := 1; i <= loginBurst+1; i++ {
		attemptLogin(s, "admin", fmt.Sprintf("203.0.113.%d", i))
	}

	// "admin" is now throttled; a different account from a fresh IP is not.
	if got := attemptLogin(s, "someone-else", "203.0.113.200"); got != http.StatusUnauthorized {
		t.Fatalf("unrelated account collaterally throttled: got %d, want 401", got)
	}
}

// The account key is case-insensitive, so ADMIN cannot dodge admin's lockout.
func TestPerAccountThrottle_caseInsensitive(t *testing.T) {
	s := newTestServer(t)

	for i := 1; i <= loginBurst; i++ {
		attemptLogin(s, "admin", fmt.Sprintf("192.0.2.%d", i))
	}
	if got := attemptLogin(s, "ADMIN", "192.0.2.99"); got != http.StatusTooManyRequests {
		t.Fatalf("case variation dodged the account throttle: got %d, want 429", got)
	}
}

// The throttle key must come from RemoteAddr, never from a forgeable header.
func TestPerIPThrottle_ignoresForgedForwardedFor(t *testing.T) {
	s := newTestServer(t)

	burn := func(xff string) int {
		body := `{"username":"u1","password":"wrongpassword123"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
		req.RemoteAddr = "203.0.113.77:40000" // same host every time
		req.Header.Set("X-Forwarded-For", xff)
		rr := httptest.NewRecorder()
		s.handleLogin(rr, req)
		return rr.Code
	}

	for i := 1; i <= loginBurst; i++ {
		if got := burn(fmt.Sprintf("10.0.0.%d", i)); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d, want 401", i, got)
		}
	}
	if got := burn("10.0.0.250"); got != http.StatusTooManyRequests {
		t.Fatalf("rotating X-Forwarded-For dodged the per-IP throttle: got %d, want 429", got)
	}
}

// TestGenerateJWT_setsUniqueJTI guards the revocation path: a token without a
// jti can never be revoked, which silently made logout a no-op.
func TestGenerateJWT_setsUniqueJTI(t *testing.T) {
	secret := "test-secret"

	first, err := GenerateJWT(1, "admin", "admin", false, secret)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	second, err := GenerateJWT(1, "admin", "admin", false, secret)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	c1, err := ValidateJWT(first, secret)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}
	c2, err := ValidateJWT(second, secret)
	if err != nil {
		t.Fatalf("ValidateJWT: %v", err)
	}

	if c1.ID == "" {
		t.Fatal("issued token has no jti — logout revocation cannot work")
	}
	if c1.ID == c2.ID {
		t.Fatalf("jti is not unique across tokens: %q", c1.ID)
	}
}

// TestLogoutRevokesToken checks the full loop: a token accepted by
// JWTMiddleware is rejected after logout.
func TestLogoutRevokesToken(t *testing.T) {
	s := newTestServer(t)

	token, err := GenerateJWT(1, "admin", "admin", false, s.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	protected := s.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	call := func(h http.HandlerFunc) int {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		h(rr, req)
		return rr.Code
	}

	if got := call(protected); got != http.StatusOK {
		t.Fatalf("before logout: got %d, want 200", got)
	}
	if got := call(s.JWTMiddleware(s.handleLogout)); got != http.StatusOK {
		t.Fatalf("logout: got %d, want 200", got)
	}
	if got := call(protected); got != http.StatusUnauthorized {
		t.Fatalf("after logout: got %d, want 401 — revoked token still accepted", got)
	}
}

// TestEmptyAdminTokenDoesNotAuthenticate guards the "" == "" bypass: when
// AdminToken is unset, a request with no Authorization header must not pass.
func TestEmptyAdminTokenDoesNotAuthenticate(t *testing.T) {
	s := newTestServer(t, WithConfig(&config.Config{
		Port:       8080,
		AdminToken: "", // blanked by config.Save(), unset in dev
		JWTSecret:  "test-secret",
	}))

	if s.isAdminToken("") {
		t.Fatal("empty token authenticated as admin")
	}

	// Every middleware that hand-rolled the admin comparison.
	guards := map[string]func(http.HandlerFunc) http.HandlerFunc{
		"authMiddleware":       s.authMiddleware,
		"adminMiddleware":      s.adminMiddleware,
		"adminTokenRoute":      s.adminTokenRoute,
		"agentOrViewerRoute":   s.agentOrViewerRoute,
		"agentOrOperatorRoute": s.agentOrOperatorRoute,
		"agentOrAdminRoute":    s.agentOrAdminRoute,
	}

	for name, guard := range guards {
		t.Run(name, func(t *testing.T) {
			reached := false
			h := guard(func(w http.ResponseWriter, r *http.Request) { reached = true })

			// No Authorization header at all.
			rr := httptest.NewRecorder()
			h(rr, httptest.NewRequest(http.MethodGet, "/", nil))

			if reached {
				t.Fatalf("%s: handler reached with no Authorization header", name)
			}
			if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
				t.Fatalf("%s: got %d, want 401 or 403", name, rr.Code)
			}
		})
	}
}
