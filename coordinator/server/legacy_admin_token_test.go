package server

import (
	"bytes"
	"log"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Machines still authenticating with the admin token must be reported once per
// host — that log is the migration list gating removal of the isAdminToken
// branch in acceptMachineToken. Agents heartbeat every 30s, so an
// undeduplicated warning would bury the log and get muted instead of acted on.
func TestWarnLegacyAdminTokenAgent_deduplicatesPerIP(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	s := &Server{}

	req := func(addr string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/agents/a1/heartbeat", nil)
		r.RemoteAddr = addr
		s.warnLegacyAdminTokenAgent(r)
		return nil
	}

	for i := 0; i < 5; i++ {
		req("10.0.0.7:5001")
	}
	req("10.0.0.8:5002")

	got := buf.String()
	if n := strings.Count(got, "10.0.0.7"); n != 1 {
		t.Errorf("first host warned %d times, want exactly 1:\n%s", n, got)
	}
	if n := strings.Count(got, "10.0.0.8"); n != 1 {
		t.Errorf("second host warned %d times, want exactly 1:\n%s", n, got)
	}
	if !strings.Contains(got, "DEPRECATED") {
		t.Errorf("warning should be greppable as DEPRECATED, got:\n%s", got)
	}
}

// The map is written from request goroutines, so it must be mutex-guarded.
// Run with -race to make this meaningful.
func TestWarnLegacyAdminTokenAgent_concurrent(t *testing.T) {
	log.SetOutput(&bytes.Buffer{})
	defer log.SetOutput(log.Writer())

	s := &Server{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := httptest.NewRequest("GET", "/api/jobs", nil)
			r.RemoteAddr = "10.0.0.9:6000"
			s.warnLegacyAdminTokenAgent(r)
		}()
	}
	wg.Wait()
}
