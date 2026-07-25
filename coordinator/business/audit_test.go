package business

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"arcvault/coordinator/db"
)

// mockAuditQueries implements db.AuditService for tests.
type mockAuditQueries struct {
	mu        sync.Mutex
	entries   []db.UserAuditLogEntry
	nextID    int
	listErr   error
	insertErr error
}

func newMockAuditQueries() *mockAuditQueries {
	return &mockAuditQueries{
		entries: []db.UserAuditLogEntry{},
		nextID:  1,
	}
}

func (m *mockAuditQueries) InsertUserAuditLog(ctx db.UserAuditLogContext) error {
	if m.insertErr != nil {
		return m.insertErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := db.UserAuditLogEntry{
		ID:        m.nextID,
		UserID:    ctx.UserID,
		Username:  ctx.Username,
		UserRole:  ctx.UserRole,
		Action:    ctx.Action,
		IPAddress: ctx.IPAddress,
		Success:   ctx.Success,
		CreatedAt: time.Now(),
	}
	if ctx.ResourceType != nil {
		rt := *ctx.ResourceType
		entry.ResourceType = &rt
	}
	if ctx.ResourceID != nil {
		ri := *ctx.ResourceID
		entry.ResourceID = &ri
	}
	if ctx.Details != nil {
		d := *ctx.Details
		entry.Details = &d
	}
	m.nextID++
	m.entries = append(m.entries, entry)
	return nil
}

func (m *mockAuditQueries) ListUserAuditLogs(filter db.UserAuditLogFilter) ([]db.UserAuditLogEntry, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return entries in reverse order (DESC).
	var result []db.UserAuditLogEntry
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := m.entries[i]
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		result = append(result, e)
	}
	total := len(result)
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			result = nil
		} else {
			result = result[filter.Offset:]
		}
	}
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	if result == nil {
		result = []db.UserAuditLogEntry{}
	}
	return result, total, nil
}

func strPtr(s string) *string { return &s }

// ---------- AuditService tests ----------

func TestAuditService_LogAction(t *testing.T) {
	m := newMockAuditQueries()
	svc := NewAuditService(m)

	userID := 1
	err := svc.LogAction(&userID, "admin", "admin", "user.create", "192.168.1.1", true, strPtr("user"), strPtr("testuser"), nil)
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}

	entries, total, err := svc.ListAuditLogs(AuditLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Action != "user.create" {
		t.Fatalf("expected action 'user.create', got %q", entries[0].Action)
	}
	if entries[0].Username != "admin" {
		t.Fatalf("expected username 'admin', got %q", entries[0].Username)
	}
	if entries[0].ResourceType == nil || *entries[0].ResourceType != "user" {
		t.Fatalf("expected resource_type 'user', got %v", entries[0].ResourceType)
	}
	if entries[0].ResourceID == nil || *entries[0].ResourceID != "testuser" {
		t.Fatalf("expected resource_id 'testuser', got %v", entries[0].ResourceID)
	}
}

func TestAuditService_LogAction_Failure(t *testing.T) {
	m := newMockAuditQueries()
	svc := NewAuditService(m)

	err := svc.LogAction(nil, "viewer", "viewer", "job.run", "10.0.0.1", false, nil, nil, strPtr("connection refused"))
	if err != nil {
		t.Fatalf("LogAction failed: %v", err)
	}

	entries, total, err := svc.ListAuditLogs(AuditLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if entries[0].Success {
		t.Fatalf("expected success=false, got true")
	}
	if entries[0].Details == nil || *entries[0].Details != "connection refused" {
		t.Fatalf("expected details 'connection refused', got %v", entries[0].Details)
	}
	if entries[0].UserID != nil {
		t.Fatalf("expected nil user_id, got %v", entries[0].UserID)
	}
}

func TestAuditService_ListAuditLogs_FilterByAction(t *testing.T) {
	m := newMockAuditQueries()
	svc := NewAuditService(m)

	svc.LogAction(nil, "admin", "admin", "user.create", "1.2.3.4", true, nil, nil, nil)
	svc.LogAction(nil, "admin", "admin", "user.delete", "1.2.3.4", true, nil, nil, nil)
	svc.LogAction(nil, "admin", "admin", "user.create", "1.2.3.4", true, nil, nil, nil)

	entries, total, err := svc.ListAuditLogs(AuditLogFilter{Action: "user.create", Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for user.create, got %d", total)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Action != "user.create" {
			t.Fatalf("expected action 'user.create', got %q", e.Action)
		}
	}
}

func TestAuditService_ListAuditLogs_Pagination(t *testing.T) {
	m := newMockAuditQueries()
	svc := NewAuditService(m)

	for i := 0; i < 25; i++ {
		svc.LogAction(nil, "admin", "admin", "action", "127.0.0.1", true, nil, nil, nil)
	}

	// Page 1
	entries, total, err := svc.ListAuditLogs(AuditLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs page 1 failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(entries) != 10 {
		t.Fatalf("expected 10 entries on page 1, got %d", len(entries))
	}

	// Page 2
	entries, total, err = svc.ListAuditLogs(AuditLogFilter{Page: 2, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs page 2 failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(entries) != 10 {
		t.Fatalf("expected 10 entries on page 2, got %d", len(entries))
	}

	// Page 3
	entries, total, err = svc.ListAuditLogs(AuditLogFilter{Page: 3, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs page 3 failed: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total 25, got %d", total)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries on page 3, got %d", len(entries))
	}
}

func TestAuditService_ListAuditLogs_Empty(t *testing.T) {
	svc := NewAuditService(newMockAuditQueries())

	entries, total, err := svc.ListAuditLogs(AuditLogFilter{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListAuditLogs failed: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

// ---------- ClientIP tests ----------

// withTrustedProxy enables proxy-header trust for the duration of a test.
func withTrustedProxy(t *testing.T) {
	t.Helper()
	SetTrustProxyHeaders(true)
	t.Cleanup(func() { SetTrustProxyHeaders(false) })
}

func TestClientIP_ForwardedFor(t *testing.T) {
	withTrustedProxy(t)
	req := httptest.NewRequest("GET", "/", nil)
	// Rightmost entry is the one our own proxy appended.
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	ip := ClientIP(req)
	if ip != "10.0.0.2" {
		t.Fatalf("expected 10.0.0.2, got %q", ip)
	}
}

func TestClientIP_ForwardedForSingle(t *testing.T) {
	withTrustedProxy(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.5")
	ip := ClientIP(req)
	if ip != "10.0.0.5" {
		t.Fatalf("expected 10.0.0.5, got %q", ip)
	}
}

// Without a configured proxy, forged headers must be ignored entirely.
func TestClientIP_IgnoresProxyHeadersByDefault(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "5.6.7.8")
	if ip := ClientIP(req); ip != "203.0.113.9" {
		t.Fatalf("forged header trusted: got %q, want 203.0.113.9", ip)
	}
}

// A client that pre-seeds X-Forwarded-For must not displace the proxy's entry.
func TestClientIP_ForwardedForSpoofPrefix(t *testing.T) {
	withTrustedProxy(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.7")
	if ip := ClientIP(req); ip != "198.51.100.7" {
		t.Fatalf("spoofed prefix won: got %q, want 198.51.100.7", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	ip := ClientIP(req)
	if ip != "192.168.1.100" {
		t.Fatalf("expected 192.168.1.100, got %q", ip)
	}
}

func TestClientIP_RealIP(t *testing.T) {
	withTrustedProxy(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "172.16.0.1")
	ip := ClientIP(req)
	if ip != "172.16.0.1" {
		t.Fatalf("expected 172.16.0.1, got %q", ip)
	}
}

func TestClientIP_RemoteAddrNoPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.20.30.40"
	ip := ClientIP(req)
	if ip != "10.20.30.40" {
		t.Fatalf("expected 10.20.30.40, got %q", ip)
	}
}
