package db

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCreateAndGetUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a user
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	err := db.CreateUser("testuser", string(hash), "operator", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Get the user
	user, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	if user == nil {
		t.Fatal("Expected user, got nil")
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}
	if user.Role != "operator" {
		t.Errorf("Expected role 'operator', got '%s'", user.Role)
	}
	if user.MustChangePassword != false {
		t.Errorf("Expected MustChangePassword false, got %v", user.MustChangePassword)
	}
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	err := db.CreateUser("user1", string(hash), "viewer", false)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := db.GetUserByUsername("user1")
	if err != nil {
		t.Fatalf("GetUserByUsername failed: %v", err)
	}

	user2, err := db.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if user2 == nil || user2.ID != user.ID {
		t.Error("GetUserByID returned wrong user")
	}
}

func TestCountUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)

	count, _ := db.CountUsers()
	if count != 1 {
		t.Errorf("Expected 1 user initially (default admin), got %d", count)
	}

	db.CreateUser("user1", string(hash), "admin", false)
	count, _ = db.CountUsers()
	if count != 2 {
		t.Errorf("Expected 2 users after create, got %d", count)
	}

	db.CreateUser("user2", string(hash), "operator", false)
	count, _ = db.CountUsers()
	if count != 3 {
		t.Errorf("Expected 3 users after second create, got %d", count)
	}
}

func TestUpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	newHash, _ := bcrypt.GenerateFromPassword([]byte("newpass"), bcrypt.DefaultCost)

	err := db.CreateUser("testuser", string(oldHash), "admin", true)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, _ := db.GetUserByUsername("testuser")
	if !user.MustChangePassword {
		t.Error("Expected MustChangePassword to be true")
	}

	err = db.UpdatePassword(user.ID, string(newHash), false)
	if err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	user, _ = db.GetUserByUsername("testuser")
	if user.MustChangePassword {
		t.Error("Expected MustChangePassword to be false after update")
	}
}

func TestListUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	db.CreateUser("user1", string(hash), "admin", false)
	db.CreateUser("user2", string(hash), "operator", false)
	db.CreateUser("user3", string(hash), "viewer", false)

	users, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 4 {
		t.Errorf("Expected 4 users (1 default admin + 3 created), got %d", len(users))
	}

	if users[0].Username != "admin" {
		t.Errorf("Expected first user to be 'admin' (default), got '%s'", users[0].Username)
	}

	if users[1].Username != "user1" {
		t.Errorf("Expected second user to be 'user1', got '%s'", users[1].Username)
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	db.CreateUser("testuser", string(hash), "admin", false)

	user, _ := db.GetUserByUsername("testuser")
	err := db.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	user, _ = db.GetUserByUsername("testuser")
	if user != nil {
		t.Error("Expected user to be deleted, but still found")
	}
}

func TestUpdateUserRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	db.CreateUser("testuser", string(hash), "viewer", false)

	user, _ := db.GetUserByUsername("testuser")
	err := db.UpdateUserRole(user.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateUserRole failed: %v", err)
	}

	user, _ = db.GetUserByUsername("testuser")
	if user.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", user.Role)
	}
}

// Helper to create test database
func setupTestDB(t *testing.T) *DB {
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	db, err := Init(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})

	return db
}
