package business

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCreateUserInput_validate(t *testing.T) {
	cases := []struct {
		input   CreateUserInput
		wantErr string
	}{
		{CreateUserInput{"", "pass", "admin"}, "username is required"},
		{CreateUserInput{"alice", "", "admin"}, "password is required"},
		{CreateUserInput{"alice", "pass", ""}, "role is required"},
		{CreateUserInput{"alice", "pass", "superuser"}, "invalid role: must be 'admin' or 'viewer'"},
	}

	for _, tc := range cases {
		err := tc.input.Validate()
		if err == nil || err.Error() != tc.wantErr {
			t.Errorf("Validate(%+v): expected %q, got %v", tc.input, tc.wantErr, err)
		}
	}
}

func TestCreateUserInput_validInputPasses(t *testing.T) {
	for _, role := range []string{"admin", "viewer"} {
		input := CreateUserInput{Username: "alice", Password: "secret", Role: role}
		if err := input.Validate(); err != nil {
			t.Errorf("expected valid for role=%q, got %v", role, err)
		}
	}
}

func TestCreateUser_duplicateUsernameReturnsError(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	input := &CreateUserInput{Username: "alice", Password: "secret", Role: "admin"}
	svc.CreateUser(input)

	_, err := svc.CreateUser(input)
	if err == nil || err.Error() != "user already exists" {
		t.Errorf("expected 'user already exists', got %v", err)
	}
}

func TestCreateUser_success(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	user, err := svc.CreateUser(&CreateUserInput{Username: "alice", Password: "secret", Role: "viewer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected Username 'alice', got %q", user.Username)
	}
	if user.Role != "viewer" {
		t.Errorf("expected Role 'viewer', got %q", user.Role)
	}
	if !user.MustChangePassword {
		t.Error("expected MustChangePassword=true for new users")
	}
}

func TestValidateCredentials_wrongPasswordFails(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	svc.CreateUser(&CreateUserInput{Username: "alice", Password: "correct", Role: "admin"})

	_, err := svc.ValidateCredentials("alice", "wrong")
	if err == nil || err.Error() != "authentication failed" {
		t.Errorf("expected 'authentication failed', got %v", err)
	}
}

func TestValidateCredentials_unknownUserFails(t *testing.T) {
	svc := NewUserService(newMockUserQueries())

	_, err := svc.ValidateCredentials("nobody", "pass")
	if err == nil || err.Error() != "authentication failed" {
		t.Errorf("expected 'authentication failed', got %v", err)
	}
}

func TestValidateCredentials_success(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	svc.CreateUser(&CreateUserInput{Username: "alice", Password: "correct", Role: "admin"})

	user, err := svc.ValidateCredentials("alice", "correct")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Errorf("expected Username 'alice', got %q", user.Username)
	}
}

func TestUpdatePassword_wrongOldPasswordFails(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	u, _ := svc.CreateUser(&CreateUserInput{Username: "alice", Password: "old", Role: "admin"})

	err := svc.UpdatePassword(u.ID, "wrong-old", "new-pass")
	if err == nil || err.Error() != "incorrect password" {
		t.Errorf("expected 'incorrect password', got %v", err)
	}
}

func TestUpdatePassword_success(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	u, _ := svc.CreateUser(&CreateUserInput{Username: "alice", Password: "old", Role: "admin"})

	if err := svc.UpdatePassword(u.ID, "old", "new-pass"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the stored hash now matches new password
	stored := mock.users["alice"]
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("new-pass")); err != nil {
		t.Error("expected stored hash to match new password")
	}
}

func TestUpdateUserRole_invalidRole(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	u, _ := svc.CreateUser(&CreateUserInput{Username: "alice", Password: "pass", Role: "admin"})

	err := svc.UpdateUserRole(u.ID, "superuser")
	if err == nil || err.Error() != "invalid role: must be 'admin' or 'viewer'" {
		t.Errorf("expected invalid role error, got %v", err)
	}
}

func TestDeleteUser_notFound(t *testing.T) {
	svc := NewUserService(newMockUserQueries())

	err := svc.DeleteUser(999)
	if err == nil || err.Error() != "user not found" {
		t.Errorf("expected 'user not found', got %v", err)
	}
}

func TestDeleteUser_success(t *testing.T) {
	mock := newMockUserQueries()
	svc := NewUserService(mock)

	u, _ := svc.CreateUser(&CreateUserInput{Username: "alice", Password: "pass", Role: "admin"})

	if err := svc.DeleteUser(u.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := mock.users["alice"]; exists {
		t.Error("expected user to be removed")
	}
}
