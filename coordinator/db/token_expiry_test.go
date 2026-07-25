package db

import (
	"regexp"
	"testing"
	"time"
)

// sqliteDatetime matches the only format that string-compares correctly against
// SQLite's datetime('now'). Go's time.Time.String() ("... -0400 EDT") does not.
var sqliteDatetime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)

// forceNonUTCLocal pins time.Local to a negative offset for the duration of the
// test. Without this the bug hides on UTC machines: it only bites when local
// time is behind UTC, which is exactly where it shipped.
func forceNonUTCLocal(t *testing.T) {
	t.Helper()
	orig := time.Local
	time.Local = time.FixedZone("EDT", -4*60*60)
	t.Cleanup(func() { time.Local = orig })
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Init(":memory:")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// A revoked token must read as revoked, and the stored expiry must be a
// canonical UTC SQLite datetime — not Go's String() form in local time.
func TestRevokeToken_storesComparableUTCExpiry(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	// 4h lifetime is what GenerateJWT issues. With a -4h local offset and the
	// old code, the stored string equalled the *issue* time in UTC, so the
	// expiry clause went false the moment the clock ticked past that second.
	jti := "abc123"
	if err := d.RevokeToken(jti, time.Now().Add(4*time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	// `expires_at || ''` defeats the driver's DATETIME->time.Time conversion, so
	// we see the bytes SQLite actually compares against datetime('now').
	var stored string
	if err := d.conn.QueryRow(`SELECT expires_at || '' FROM revoked_tokens WHERE jti = ?`, jti).Scan(&stored); err != nil {
		t.Fatalf("read back expires_at: %v", err)
	}
	if !sqliteDatetime.MatchString(stored) {
		t.Fatalf("expires_at = %q; want canonical UTC 'YYYY-MM-DD HH:MM:SS' — "+
			"a Go-formatted timestamp sorts below datetime('now') and defeats the expiry clause", stored)
	}

	revoked, err := d.IsTokenRevoked(jti)
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("revoked token reads as live — logout does not revoke")
	}
}

// Prune must drop expired rows and keep live ones. Getting this backwards
// un-revokes every logged-out token, so assert both directions, not just the count.
func TestPruneExpiredTokens_dropsExpiredKeepsLive(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	if err := d.RevokeToken("expired", time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatalf("RevokeToken(expired): %v", err)
	}
	if err := d.RevokeToken("live", time.Now().Add(4*time.Hour)); err != nil {
		t.Fatalf("RevokeToken(live): %v", err)
	}

	if err := d.PruneExpiredTokens(); err != nil {
		t.Fatalf("PruneExpiredTokens: %v", err)
	}

	var expiredRows int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM revoked_tokens WHERE jti = 'expired'`).Scan(&expiredRows); err != nil {
		t.Fatalf("count expired: %v", err)
	}
	if expiredRows != 0 {
		t.Error("expired row survived prune — revoked_tokens grows without bound")
	}

	// The live row must still be *functionally* revoked, not merely present.
	revoked, err := d.IsTokenRevoked("live")
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("prune un-revoked a live token — a logged-out session is valid again")
	}
}

// The old bug's tell: it only passed while the stored string and datetime('now')
// shared a second-level prefix. Let the clock tick past that and it flips.
func TestRevokeToken_survivesClockTick(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	jti := "tick-test"
	if err := d.RevokeToken(jti, time.Now().Add(4*time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	time.Sleep(1100 * time.Millisecond) // guarantee datetime('now') advances a second

	revoked, err := d.IsTokenRevoked(jti)
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if !revoked {
		t.Fatal("revocation decayed after one second — expiry stored in the wrong zone/format")
	}
}

// An expired revocation entry should stop matching; PruneExpiredTokens removes it.
func TestRevokeToken_expiredEntryIsNotRevoked(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	jti := "already-expired"
	if err := d.RevokeToken(jti, time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
	revoked, err := d.IsTokenRevoked(jti)
	if err != nil {
		t.Fatalf("IsTokenRevoked: %v", err)
	}
	if revoked {
		t.Fatal("an expired revocation entry should no longer match")
	}
	if err := d.PruneExpiredTokens(); err != nil {
		t.Fatalf("PruneExpiredTokens: %v", err)
	}
	var n int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM revoked_tokens WHERE jti = ?`, jti).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("expired entry not pruned: %d rows remain", n)
	}
}

// Same root cause, other call site: a freshly minted bootstrap token has a
// 1-hour expiry and must validate immediately.
func TestCreateAgentToken_bootstrapTokenValidatesImmediately(t *testing.T) {
	forceNonUTCLocal(t)
	d := newTestDB(t)

	tok, err := d.CreateAgentToken("bootstrap-agent-1")
	if err != nil {
		t.Fatalf("CreateAgentToken: %v", err)
	}
	if _, err := d.ValidateToken(tok); err != nil {
		t.Fatalf("fresh bootstrap token rejected: %v — expiry stored in local time reads as already past", err)
	}
}
