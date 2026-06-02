package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-testing-only"

func TestGenerateAndValidateJWT(t *testing.T) {
	tokenString, err := GenerateJWT(1, "admin", "admin", false, testSecret)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	if tokenString == "" {
		t.Fatal("Expected token, got empty string")
	}

	// Validate the token
	claims, err := ValidateJWT(tokenString, testSecret)
	if err != nil {
		t.Fatalf("ValidateJWT failed: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Expected Username 'admin', got '%s'", claims.Username)
	}
	if claims.Role != "admin" {
		t.Errorf("Expected Role 'admin', got '%s'", claims.Role)
	}
	if claims.MustChange != false {
		t.Errorf("Expected MustChange false, got %v", claims.MustChange)
	}
}

func TestJWTExpiry(t *testing.T) {
	// Create a token with custom expiry (in the past for testing)
	claims := &JWTClaims{
		UserID:    1,
		Username:  "testuser",
		Role:      "operator",
		MustChange: false,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(testSecret))

	// Should fail validation due to expiry
	_, err := ValidateJWT(tokenString, testSecret)
	if err == nil {
		t.Fatal("Expected validation to fail for expired token")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	tokenString, _ := GenerateJWT(1, "admin", "admin", false, testSecret)

	// Try to validate with wrong secret
	_, err := ValidateJWT(tokenString, "wrong-secret")
	if err == nil {
		t.Fatal("Expected validation to fail with wrong secret")
	}
}

func TestRequireRoleMiddleware(t *testing.T) {
	// Create a test handler
	handler := RequireRole("admin")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Test 1: Admin role allowed
	claims := &JWTClaims{
		UserID:   1,
		Username: "admin",
		Role:     "admin",
	}
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w := httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d for admin role", w.Code)
	}

	// Test 2: Operator role denied
	claims.Role = "operator"
	req = httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w = httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d for operator role", w.Code)
	}

	// Test 3: No claims in context
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d for missing claims", w.Code)
	}
}

func TestRequireMultipleRoles(t *testing.T) {
	// Handler that accepts admin or operator
	handler := RequireRole("admin", "operator")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Test admin role
	claims := &JWTClaims{Role: "admin"}
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for admin, got %d", w.Code)
	}

	// Test operator role
	claims.Role = "operator"
	req = httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for operator, got %d", w.Code)
	}

	// Test viewer role (denied)
	claims.Role = "viewer"
	req = httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for viewer, got %d", w.Code)
	}
}

func TestRequirePasswordChangeMiddleware(t *testing.T) {
	// Handler that checks password change requirement
	handler := RequirePasswordChange("/api/auth/change-password")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Test 1: User with must_change=true on exempt path
	claims := &JWTClaims{
		UserID:     1,
		Username:   "admin",
		Role:       "admin",
		MustChange: true,
	}
	req := httptest.NewRequest("GET", "/api/auth/change-password", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w := httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 on exempt path, got %d", w.Code)
	}

	// Test 2: User with must_change=true on non-exempt path
	req = httptest.NewRequest("GET", "/api/jobs", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w = httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 when must_change=true, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "must_change_password" {
		t.Errorf("Expected error 'must_change_password', got '%s'", resp["error"])
	}

	// Test 3: User with must_change=false can access any path
	claims.MustChange = false
	req = httptest.NewRequest("GET", "/api/jobs", nil)
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	w = httptest.NewRecorder()

	handler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 when must_change=false, got %d", w.Code)
	}
}

func TestGetUserClaims(t *testing.T) {
	claims := &JWTClaims{
		UserID:   1,
		Username: "testuser",
		Role:     "admin",
	}

	req := httptest.NewRequest("GET", "/test", nil)

	// Test without claims in context
	retrieved := GetUserClaims(req)
	if retrieved != nil {
		t.Error("Expected nil for missing claims")
	}

	// Test with claims in context
	req = req.WithContext(contextWithClaims(req.Context(), claims))
	retrieved = GetUserClaims(req)
	if retrieved == nil {
		t.Fatal("Expected claims, got nil")
	}
	if retrieved.UserID != claims.UserID {
		t.Errorf("Expected UserID %d, got %d", claims.UserID, retrieved.UserID)
	}
}

// Helper to add claims to context
func contextWithClaims(ctx context.Context, claims *JWTClaims) context.Context {
	return context.WithValue(ctx, UserClaimsCtxKey{}, claims)
}

// === Endpoint Handler Tests ===

func TestLogin_successWithValidCredentials(t *testing.T) {
	// This test requires a database and user setup, so it would normally be in an integration test
	// For now, we're testing the JWT generation/validation logic above
	// Full endpoint tests would require setting up a test server with DB
}

func TestBcryptHash(t *testing.T) {
	password := "testpassword"
	hash, err := bcryptHash(password)
	if err != nil {
		t.Fatalf("bcryptHash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Expected hash, got empty string")
	}

	// Verify hash
	err = bcryptCompare(hash, password)
	if err != nil {
		t.Fatalf("bcryptCompare failed: %v", err)
	}
}

func TestBcryptCompare_failsWithWrongPassword(t *testing.T) {
	password := "correctpassword"
	hash, _ := bcryptHash(password)

	err := bcryptCompare(hash, "wrongpassword")
	if err == nil {
		t.Fatal("Expected bcryptCompare to fail with wrong password")
	}
}
