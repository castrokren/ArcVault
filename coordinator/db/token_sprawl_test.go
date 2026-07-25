package db

import "testing"

// countTokens returns how many rows exist for an agent_id.
func countTokens(t *testing.T, d *DB, agentID string) int {
	t.Helper()
	var n int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM tokens WHERE agent_id = ?`, agentID).Scan(&n); err != nil {
		t.Fatalf("count tokens for %q: %v", agentID, err)
	}
	return n
}

// An agent's tokens never expire and nothing pruned the table, so every mint used
// to leave a permanently valid credential behind. rebuild-and-restart.ps1 mints on
// every deploy, which had accumulated 12 live tokens for one agent on the live box.
func TestCreateAgentToken_supersedesPreviousTokensForSameAgent(t *testing.T) {
	d := newTestDB(t)

	first, err := d.CreateAgentToken("AGENT-1")
	if err != nil {
		t.Fatalf("first mint: %v", err)
	}
	second, err := d.CreateAgentToken("AGENT-1")
	if err != nil {
		t.Fatalf("second mint: %v", err)
	}

	if n := countTokens(t, d, "AGENT-1"); n != 1 {
		t.Errorf("expected exactly 1 live token after re-mint, got %d", n)
	}
	if _, err := d.ValidateToken(second); err != nil {
		t.Errorf("newest token must validate: %v", err)
	}
	if _, err := d.ValidateToken(first); err == nil {
		t.Error("superseded token still validates — the old credential remains usable")
	}
}

// Superseding must be scoped to one agent. Deleting by anything broader would log
// out the rest of the fleet on the next deploy.
func TestCreateAgentToken_doesNotTouchOtherAgents(t *testing.T) {
	d := newTestDB(t)

	other, err := d.CreateAgentToken("AGENT-2")
	if err != nil {
		t.Fatalf("mint for AGENT-2: %v", err)
	}
	if _, err := d.CreateAgentToken("AGENT-1"); err != nil {
		t.Fatalf("mint for AGENT-1: %v", err)
	}

	if _, err := d.ValidateToken(other); err != nil {
		t.Errorf("AGENT-2's token was revoked by a mint for AGENT-1: %v", err)
	}
	if n := countTokens(t, d, "AGENT-2"); n != 1 {
		t.Errorf("AGENT-2 should still have 1 token, got %d", n)
	}
}

// Enrollment tokens are NOT superseded: several machines can be mid-enrollment at
// once, and generating a second install script must not invalidate the first.
func TestCreateAgentToken_enrollmentTokensCoexist(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	a, err := d.CreateAgentToken("bootstrap:HOST-A")
	if err != nil {
		t.Fatalf("mint for HOST-A: %v", err)
	}
	b, err := d.CreateAgentToken("bootstrap:HOST-B")
	if err != nil {
		t.Fatalf("mint for HOST-B: %v", err)
	}
	c, err := d.CreateAgentToken("bootstrap")
	if err != nil {
		t.Fatalf("mint for plain bootstrap: %v", err)
	}
	d2, err := d.CreateAgentToken("bootstrap")
	if err != nil {
		t.Fatalf("second mint for plain bootstrap: %v", err)
	}

	for name, tok := range map[string]string{"HOST-A": a, "HOST-B": b, "bootstrap#1": c, "bootstrap#2": d2} {
		if _, err := d.ValidateToken(tok); err != nil {
			t.Errorf("enrollment token %s should still validate: %v", name, err)
		}
	}
}

func TestIsEnrollmentToken(t *testing.T) {
	cases := map[string]bool{
		"bootstrap":            true,
		"bootstrap:HOST-A":     true,
		"DESKTOP-EE77F38":      false,
		"":                     false,
		"my-bootstrap-machine": false, // prefix match, not substring
	}
	for in, want := range cases {
		if got := IsEnrollmentToken(in); got != want {
			t.Errorf("IsEnrollmentToken(%q) = %v, want %v", in, got, want)
		}
	}
}

// The tokens-table half of the pruner (the revoked_tokens half is covered by
// TestPruneExpiredTokens_dropsExpiredKeepsLive in token_expiry_test.go).
// GC only: drop expired rows, leave live ones — including per-agent tokens, whose
// NULL expires_at means "does not expire". forceNonUTCLocal because this compares
// stored timestamps against datetime('now'), the exact spot the local-vs-UTC bug bit.
func TestPruneExpiredTokens_dropsExpiredAgentTokens(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	agentTok, err := d.CreateAgentToken("AGENT-1")
	if err != nil {
		t.Fatalf("mint agent token: %v", err)
	}
	enrollTok, err := d.CreateAgentToken("bootstrap:HOST-A")
	if err != nil {
		t.Fatalf("mint enrollment token: %v", err)
	}

	// Backdate the enrollment token past its expiry.
	if _, err := d.conn.Exec(
		`UPDATE tokens SET expires_at = datetime('now', '-1 hour') WHERE token = ?`, enrollTok,
	); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	if err := d.PruneExpiredTokens(); err != nil {
		t.Fatalf("PruneExpiredTokens: %v", err)
	}

	if n := countTokens(t, d, "bootstrap:HOST-A"); n != 0 {
		t.Errorf("expired enrollment token not pruned: %d rows remain", n)
	}
	if _, err := d.ValidateToken(agentTok); err != nil {
		t.Errorf("pruner deleted a live non-expiring agent token: %v", err)
	}
}
