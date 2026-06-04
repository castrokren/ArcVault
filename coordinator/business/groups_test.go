package business

import (
	"testing"
)

func TestCreateGroupInput_validate(t *testing.T) {
	err := (&CreateGroupInput{Name: ""}).Validate()
	if err == nil || err.Error() != "name is required" {
		t.Errorf("expected 'name is required', got %v", err)
	}

	if err := (&CreateGroupInput{Name: "my-group"}).Validate(); err != nil {
		t.Errorf("expected valid input to pass, got %v", err)
	}
}

func TestCreateGroup_success(t *testing.T) {
	svc := NewGroupService(newMockGroupQueries())

	group, err := svc.CreateGroup(&CreateGroupInput{Name: "ops-team", Description: "ops"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "ops-team" {
		t.Errorf("expected Name 'ops-team', got %q", group.Name)
	}
	if group.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if group.AgentCount != 0 {
		t.Errorf("expected AgentCount=0 for new group, got %d", group.AgentCount)
	}
}

func TestGetGroup_notFound(t *testing.T) {
	svc := NewGroupService(newMockGroupQueries())

	_, err := svc.GetGroup(999)
	if err == nil || err.Error() != "group not found" {
		t.Errorf("expected 'group not found', got %v", err)
	}
}

func TestGetGroup_agentCountReflectsMembers(t *testing.T) {
	mock := newMockGroupQueries()
	svc := NewGroupService(mock)

	g, _ := svc.CreateGroup(&CreateGroupInput{Name: "team"})
	mock.members[g.ID] = []string{"agent-a", "agent-b"}

	got, err := svc.GetGroup(g.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AgentCount != 2 {
		t.Errorf("expected AgentCount=2, got %d", got.AgentCount)
	}
}

func TestUpdateGroup_notFound(t *testing.T) {
	svc := NewGroupService(newMockGroupQueries())

	_, err := svc.UpdateGroup(999, &UpdateGroupInput{Name: "new-name"})
	if err == nil || err.Error() != "group not found" {
		t.Errorf("expected 'group not found', got %v", err)
	}
}

func TestUpdateGroup_emptyNameFails(t *testing.T) {
	mock := newMockGroupQueries()
	svc := NewGroupService(mock)

	g, _ := svc.CreateGroup(&CreateGroupInput{Name: "original"})

	_, err := svc.UpdateGroup(g.ID, &UpdateGroupInput{Name: ""})
	if err == nil || err.Error() != "name is required" {
		t.Errorf("expected 'name is required', got %v", err)
	}
}

func TestDeleteGroup_notFound(t *testing.T) {
	svc := NewGroupService(newMockGroupQueries())

	err := svc.DeleteGroup(999)
	if err == nil || err.Error() != "group not found" {
		t.Errorf("expected 'group not found', got %v", err)
	}
}

func TestDeleteGroup_success(t *testing.T) {
	mock := newMockGroupQueries()
	svc := NewGroupService(mock)

	g, _ := svc.CreateGroup(&CreateGroupInput{Name: "team"})

	if err := svc.DeleteGroup(g.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := mock.groups[g.ID]; exists {
		t.Error("expected group to be removed")
	}
}

func TestAddAgentToGroup_groupNotFound(t *testing.T) {
	svc := NewGroupService(newMockGroupQueries())

	err := svc.AddAgentToGroup(999, "agent-01")
	if err == nil || err.Error() != "group not found" {
		t.Errorf("expected 'group not found', got %v", err)
	}
}

func TestListGroups_agentCountsPopulated(t *testing.T) {
	mock := newMockGroupQueries()
	svc := NewGroupService(mock)

	g1, _ := svc.CreateGroup(&CreateGroupInput{Name: "g1"})
	g2, _ := svc.CreateGroup(&CreateGroupInput{Name: "g2"})
	mock.members[g1.ID] = []string{"a", "b", "c"}
	mock.members[g2.ID] = []string{"d"}

	groups, err := svc.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Name] = g.AgentCount
	}
	if counts["g1"] != 3 {
		t.Errorf("expected g1 AgentCount=3, got %d", counts["g1"])
	}
	if counts["g2"] != 1 {
		t.Errorf("expected g2 AgentCount=1, got %d", counts["g2"])
	}
}
