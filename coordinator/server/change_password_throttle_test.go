package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A stolen token must not allow unlimited old-password guesses: repeated wrong
// old passwords for one user must start returning 429 once the burst is spent.
func TestChangePassword_ThrottlesBruteForce(t *testing.T) {
	s := newTestServer(t)
	claims := &JWTClaims{UserID: 1, Username: "admin", Role: "admin"}

	attempt := func(userID int) int {
		c := &JWTClaims{UserID: userID, Username: "admin", Role: "admin"}
		body := strings.NewReader(`{"old_password":"wrong-guess","new_password":"newpassword123"}`)
		req := httptest.NewRequest(http.MethodPut, "/api/auth/change-password", body)
		req = req.WithContext(contextWithClaims(req.Context(), c))
		w := httptest.NewRecorder()
		s.handleChangePassword(w, req)
		return w.Code
	}

	// loginBurst (5) attempts are allowed through to the password check; the
	// next must be throttled (429) rather than reaching bcrypt again.
	throttled := false
	for i := 0; i < loginBurst+1; i++ {
		if attempt(claims.UserID) == http.StatusTooManyRequests {
			throttled = true
			break
		}
	}
	if !throttled {
		t.Fatalf("expected a 429 within %d attempts, never throttled", loginBurst+1)
	}

	// A different user is keyed separately and must still get through.
	if code := attempt(2); code == http.StatusTooManyRequests {
		t.Fatalf("different user was throttled by user 1's attempts (code %d)", code)
	}
}
