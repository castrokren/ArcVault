package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// `host` is optional in config.json and the installer never writes it. The URL was
// built as fmt.Sprintf("https://%s", cfg.Host), which yielded the literal string
// "https://" — every curl in the generated script had nowhere to go, so a machine
// ran the installer and silently never registered. Seen live 2026-07-25: a
// bootstrap.ps1 download succeeded, and no register attempt ever arrived.
func TestCoordinatorBaseURL(t *testing.T) {
	cases := []struct {
		name        string
		cfgHost     string
		cfgPort     int
		requestHost string
		want        string
		wantErr     string
	}{
		{"configured host on 443", "arcvault.lan", 443, "localhost", "https://arcvault.lan", ""},
		{"configured host, custom port", "arcvault.lan", 8443, "localhost", "https://arcvault.lan:8443", ""},
		{"configured IP", "192.168.1.10", 443, "", "https://192.168.1.10", ""},

		// The regression: no host configured.
		{"falls back to Host header", "", 443, "arcvault.corp.example", "https://arcvault.corp.example", ""},
		{"Host header carries its port", "", 443, "arcvault.corp.example:8443", "https://arcvault.corp.example:8443", ""},

		// Never emit "https://" — that was the actual bug.
		{"nothing to go on", "", 443, "", "", "set \"host\" in config.json"},

		// Loopback: right for the browser, useless on the enrolled machine.
		{"loopback name via header", "", 443, "localhost", "", "cannot reach that address"},
		{"loopback name with port", "", 443, "localhost:443", "", "cannot reach that address"},
		{"loopback v4 via header", "", 443, "127.0.0.1", "", "cannot reach that address"},
		{"loopback v6 via header", "", 443, "[::1]:443", "", "cannot reach that address"},
		{"loopback configured explicitly", "localhost", 443, "arcvault.lan", "", "cannot reach that address"},
	}

	for _, c := range cases {
		got, err := coordinatorBaseURL(c.cfgHost, c.cfgPort, c.requestHost)
		if c.wantErr != "" {
			if err == nil {
				t.Errorf("%s: expected an error, got %q", c.name, got)
			} else if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s: error %q does not mention %q", c.name, err, c.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
		if got == "https://" {
			t.Errorf("%s: produced the empty-host URL", c.name)
		}
	}
}

// End-to-end through the router: a loopback-only coordinator must refuse rather
// than hand back an unusable script — and must not mint a token on the way out.
func TestBootstrapScript_refusesLoopbackAndMintsNothing(t *testing.T) {
	s := newTestServer(t)
	s.cfg.Host = ""
	s.cfg.Port = 443

	before := countRows(t, s, "SELECT COUNT(*) FROM tokens")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/bootstrap.ps1?hostname=WORKSTATION01", nil)
	req.Host = "localhost"
	req.Header.Set("Authorization", authHeader())
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "host") {
		t.Errorf("error should tell the operator to set \"host\": %s", rr.Body.String())
	}
	if after := countRows(t, s, "SELECT COUNT(*) FROM tokens"); after != before {
		t.Errorf("a refused request still minted a token (%d -> %d)", before, after)
	}
}

func countRows(t *testing.T, s *Server, query string) int {
	t.Helper()
	var n int
	if err := s.db.Conn().QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}
