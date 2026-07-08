package db

import (
	"testing"
)

func TestCreateAndGetGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, err := db.CreateGroup("prod", "Production agents")
	if err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	if group == nil {
		t.Fatal("Expected group, got nil")
	}
	if group.Name != "prod" {
		t.Errorf("Expected name 'prod', got '%s'", group.Name)
	}
	if group.Description != "Production agents" {
		t.Errorf("Expected description 'Production agents', got '%s'", group.Description)
	}

	// Get by ID
	retrieved, err := db.GetGroup(group.ID)
	if err != nil {
		t.Fatalf("GetGroup failed: %v", err)
	}

	if retrieved == nil || retrieved.ID != group.ID {
		t.Error("GetGroup returned wrong group")
	}
}

func TestListGroups(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	db.CreateGroup("prod", "Production")
	db.CreateGroup("staging", "Staging")
	db.CreateGroup("dev", "Development")

	groups, err := db.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(groups))
	}
}

func TestUpdateGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production agents")

	err := db.UpdateGroup(group.ID, "prod-updated", "Updated description")
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	updated, _ := db.GetGroup(group.ID)
	if updated.Name != "prod-updated" {
		t.Errorf("Expected name 'prod-updated', got '%s'", updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}
}

func TestDeleteGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production")

	err := db.DeleteGroup(group.ID)
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	retrieved, _ := db.GetGroup(group.ID)
	if retrieved != nil {
		t.Error("Expected group to be deleted, but still found")
	}
}

func TestAddAgentToGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production")

	err := db.AddAgentToGroup(group.ID, "agent-1")
	if err != nil {
		t.Fatalf("AddAgentToGroup failed: %v", err)
	}

	members, err := db.GetGroupMembers(group.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers failed: %v", err)
	}

	if len(members) != 1 || members[0] != "agent-1" {
		t.Errorf("Expected agent-1 in group members, got %v", members)
	}
}

func TestRemoveAgentFromGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production")
	db.AddAgentToGroup(group.ID, "agent-1")
	db.AddAgentToGroup(group.ID, "agent-2")

	err := db.RemoveAgentFromGroup(group.ID, "agent-1")
	if err != nil {
		t.Fatalf("RemoveAgentFromGroup failed: %v", err)
	}

	members, _ := db.GetGroupMembers(group.ID)
	if len(members) != 1 || members[0] != "agent-2" {
		t.Errorf("Expected only agent-2 remaining, got %v", members)
	}
}

func TestGetGroupMembers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production")

	members, _ := db.GetGroupMembers(group.ID)
	if len(members) != 0 {
		t.Errorf("Expected empty members initially, got %v", members)
	}

	db.AddAgentToGroup(group.ID, "agent-1")
	db.AddAgentToGroup(group.ID, "agent-2")
	db.AddAgentToGroup(group.ID, "agent-3")

	members, _ = db.GetGroupMembers(group.ID)
	if len(members) != 3 {
		t.Errorf("Expected 3 members, got %d", len(members))
	}
}

func TestGetAgentGroup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	group, _ := db.CreateGroup("prod", "Production")
	db.AddAgentToGroup(group.ID, "agent-1")

	retrieved, err := db.GetAgentGroup("agent-1")
	if err != nil {
		t.Fatalf("GetAgentGroup failed: %v", err)
	}

	if retrieved == nil || retrieved.ID != group.ID {
		t.Error("GetAgentGroup returned wrong group")
	}

	// Test ungrouped agent
	ungrouped, _ := db.GetAgentGroup("agent-2")
	if ungrouped != nil {
		t.Error("Expected nil for ungrouped agent, got group")
	}
}

func TestGroupMembershipCascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Enable foreign keys for this test to verify cascade deletes work
	db.Conn().Exec("PRAGMA foreign_keys = ON")

	group, _ := db.CreateGroup("prod", "Production")
	db.AddAgentToGroup(group.ID, "agent-1")
	db.AddAgentToGroup(group.ID, "agent-2")

	// Delete group should cascade delete memberships
	db.DeleteGroup(group.ID)

	members, _ := db.GetGroupMembers(group.ID)
	if len(members) != 0 {
		t.Error("Expected memberships to be deleted with group")
	}
}
