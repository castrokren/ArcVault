package db

import (
	"strings"
	"testing"
)

// The DSN must use modernc.org/sqlite's `_pragma=name(value)` syntax. It once
// used mattn/go-sqlite3's `_busy_timeout=`/`_journal_mode=` keys, which modernc
// SILENTLY IGNORES — so the DB ran in default rollback-journal mode with a 0
// busy timeout, and any concurrent access returned SQLITE_BUSY. That made the
// fail-closed token-revocation check 401 on a lock blip → users were logged out
// on tab clicks. This asserts both pragmas actually take effect after Init.
func TestInit_appliesWALandBusyTimeout(t *testing.T) {
	// WAL needs a real file, not :memory: (an in-memory DB reports journal_mode=memory).
	d, err := Init(t.TempDir() + "/arcvault.db")
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	var jm string
	if err := d.conn.QueryRow("PRAGMA journal_mode").Scan(&jm); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if !strings.EqualFold(jm, "wal") {
		t.Fatalf("journal_mode=%q, want wal — DSN pragma params are being ignored by the driver again", jm)
	}

	var bt int
	if err := d.conn.QueryRow("PRAGMA busy_timeout").Scan(&bt); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if bt != 5000 {
		t.Fatalf("busy_timeout=%d, want 5000 — concurrent access will return SQLITE_BUSY and log users out", bt)
	}
}
