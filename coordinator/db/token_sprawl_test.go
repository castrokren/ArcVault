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

// REGRESSION GUARD. Minting must not revoke anything. An earlier version deleted
// the agent's other tokens inside CreateAgentToken, so clicking "Get Token" in the
// dashboard revoked the running agent's credential -- it 401'd on every request
// from that moment until agent-config.yaml was hand-edited. Observed live
// 2026-07-25 11:42:01 (mint) -> 11:42:08 (first 401).
func TestCreateAgentToken_mintDoesNotRevokeTheRunningAgentsToken(t *testing.T) {
	d := newTestDB(t)

	inUse, err := d.CreateAgentToken("AGENT-1")
	if err != nil {
		t.Fatalf("first mint: %v", err)
	}
	readOut, err := d.CreateAgentToken("AGENT-1") // operator clicks "Get Token"
	if err != nil {
		t.Fatalf("second mint: %v", err)
	}

	if _, err := d.ValidateToken(inUse); err != nil {
		t.Errorf("the running agent's token stopped working after a mint: %v", err)
	}
	if _, err := d.ValidateToken(readOut); err != nil {
		t.Errorf("newly minted token does not validate: %v", err)
	}
	if n := countTokens(t, d, "AGENT-1"); n != 2 {
		t.Errorf("expected both tokens to coexist, got %d", n)
	}
}

// Cleanup happens only once an agent has proven which token it holds.
func TestSupersedeAgentTokens_keepsTheLiveTokenDropsTheRest(t *testing.T) {
	d := newTestDB(t)

	old1, _ := d.CreateAgentToken("AGENT-1")
	old2, _ := d.CreateAgentToken("AGENT-1")
	live, err := d.CreateAgentToken("AGENT-1")
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	n, err := d.SupersedeAgentTokens("AGENT-1", live)
	if err != nil {
		t.Fatalf("SupersedeAgentTokens: %v", err)
	}
	if n != 2 {
		t.Errorf("removed %d tokens, want 2", n)
	}
	if _, err := d.ValidateToken(live); err != nil {
		t.Errorf("live token was removed: %v", err)
	}
	for i, tok := range []string{old1, old2} {
		if _, err := d.ValidateToken(tok); err == nil {
			t.Errorf("superseded token %d still validates", i)
		}
	}
}

// An empty keepToken must be a no-op, never "delete everything".
func TestSupersedeAgentTokens_emptyKeepTokenIsNoOp(t *testing.T) {
	d := newTestDB(t)

	tok, _ := d.CreateAgentToken("AGENT-1")
	if n, err := d.SupersedeAgentTokens("AGENT-1", ""); err != nil || n != 0 {
		t.Errorf("got (%d, %v), want (0, nil)", n, err)
	}
	if _, err := d.ValidateToken(tok); err != nil {
		t.Errorf("no-op call deleted the token: %v", err)
	}
}

// Superseding must be scoped to one agent, or a registration would log out the
// rest of the fleet.
func TestSupersedeAgentTokens_doesNotTouchOtherAgents(t *testing.T) {
	d := newTestDB(t)

	other, err := d.CreateAgentToken("AGENT-2")
	if err != nil {
		t.Fatalf("mint for AGENT-2: %v", err)
	}
	live, _ := d.CreateAgentToken("AGENT-1")
	if _, err := d.SupersedeAgentTokens("AGENT-1", live); err != nil {
		t.Fatalf("supersede: %v", err)
	}

	if _, err := d.ValidateToken(other); err != nil {
		t.Errorf("AGENT-2's token was revoked by AGENT-1's cleanup: %v", err)
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

// PruneBootstrapTokens is an explicit operator action (unlike the automatic
// PruneExpiredTokens), so it must remove every bootstrap-tagged row regardless
// of expiry, leave real per-agent tokens untouched, and report which agent_id
// hints it deleted so the caller can warn about hosts that never re-enrolled.
func TestPruneBootstrapTokens_removesAllBootstrapRegardlessOfExpiry(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	// A live enrollment token (not yet expired).
	live, err := d.CreateAgentToken("bootstrap:HOST-A")
	if err != nil {
		t.Fatalf("mint live bootstrap: %v", err)
	}
	// A second host's enrollment token.
	if _, err := d.CreateAgentToken("bootstrap:HOST-B"); err != nil {
		t.Fatalf("mint second bootstrap: %v", err)
	}
	// A plain (no hostname hint) enrollment token.
	if _, err := d.CreateAgentToken("bootstrap"); err != nil {
		t.Fatalf("mint plain bootstrap: %v", err)
	}
	// A real per-agent token that must survive.
	agentTok, err := d.CreateAgentToken("REAL-AGENT-1")
	if err != nil {
		t.Fatalf("mint real agent token: %v", err)
	}

	hints, err := d.PruneBootstrapTokens()
	if err != nil {
		t.Fatalf("PruneBootstrapTokens: %v", err)
	}

	wantHints := map[string]bool{"bootstrap:HOST-A": true, "bootstrap:HOST-B": true, "bootstrap": true}
	if len(hints) != len(wantHints) {
		t.Errorf("hints = %v, want 3 entries covering %v", hints, wantHints)
	}
	for _, h := range hints {
		if !wantHints[h] {
			t.Errorf("unexpected hint %q returned", h)
		}
	}

	if _, err := d.ValidateToken(live); err == nil {
		t.Error("a live (unexpired) bootstrap token survived pruning — should be gone regardless of expiry")
	}
	if _, err := d.ValidateToken(agentTok); err != nil {
		t.Errorf("real per-agent token was deleted by bootstrap pruning: %v", err)
	}
}

func TestPruneBootstrapTokens_emptyIsNoOp(t *testing.T) {
	d := newTestDB(t)
	hints, err := d.PruneBootstrapTokens()
	if err != nil {
		t.Fatalf("PruneBootstrapTokens on empty table: %v", err)
	}
	if len(hints) != 0 {
		t.Errorf("hints = %v, want empty", hints)
	}
}
