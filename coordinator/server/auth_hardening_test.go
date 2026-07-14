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
	body := fmt.Sprintf(`{"username":%q,"password":"Wrongpassword123!"}`, username)
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
		body := `{"username":"u1","password":"Wrongpassword123!"}`
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

// TestLogin_weakPasswordReturns401 ensures legacy users whose passwords don't
// meet current complexity rules can still log in. Before the fix they got a
// 400 (validation error); after the fix the login reaches bcrypt and fails
// with 401 (wrong password path).
func TestLogin_weakPasswordReturns401(t *testing.T) {
	s := newTestServer(t)

	// "Password123" has uppercase + lowercase + digit, but NO special character.
	// It fails validatePasswordStrength, but LoginRequest.Validate() should
	// now accept it (non-empty checks only) and let bcrypt decide.
	body := `{"username":"admin","password":"Password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.handleLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("legacy-format password: expected 401 (bad credentials), got %d — "+
			"login should fail on bcrypt, not validation", rr.Code)
	}
}

// TestChangePassword_weakOldAllowed verifies that a user with a legacy weak
// password can change it to a strong one. OldPassword is only checked for
// non-empty, not complexity.
func TestChangePassword_weakOldAllowed(t *testing.T) {
	s := newTestServer(t)

	// Generate a JWT for the admin user (user_id=1) — the actual password in
	// the test DB is "changeme", which itself is weak (no uppercase, no digit,
	// no special).
	token, err := GenerateJWT(1, "admin", "admin", true, s.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	changePassword := s.JWTMiddleware(s.handleChangePassword)

	// Old password "changeme" is weak but should be accepted (non-empty).
	// New password "StrongPass123!" passes all complexity rules.
	body := `{"old_password":"changeme","new_password":"StrongPass123!"}`
	req := httptest.NewRequest(http.MethodPut, "/api/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	changePassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("weak old → strong new: expected 200, got %d", rr.Code)
	}
}

// TestChangePassword_weakNewReturns400 verifies that NewPassword still
// requires full complexity — a weak new password is rejected with 400.
func TestChangePassword_weakNewReturns400(t *testing.T) {
	s := newTestServer(t)

	token, err := GenerateJWT(1, "admin", "admin", true, s.cfg.JWTSecret)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	changePassword := s.JWTMiddleware(s.handleChangePassword)

	// New password "weak" is too short and lacks required character classes.
	body := `{"old_password":"changeme","new_password":"weak"}`
	req := httptest.NewRequest(http.MethodPut, "/api/auth/change-password", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	changePassword(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("weak new password: expected 400, got %d", rr.Code)
	}
}
