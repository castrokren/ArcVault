package server

import (
	"strings"
	"testing"
)

// The hostname hint is persisted as a bootstrap token's agent_id, so it must
// stay bounded and free of separator characters.
func TestValidHostnameHint(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty is allowed (hint is optional)", "", true},
		{"plain hostname", "WORKSTATION01", true},
		{"fqdn", "host-01.corp.example.com", true},
		{"max length", strings.Repeat("a", 253), true},
		{"too long", strings.Repeat("a", 254), false},
		{"colon would forge a role prefix", "host:bootstrap", false},
		{"whitespace", "host 01", false},
		{"slash", "host/../admin", false},
		{"newline", "host\nX", false},
		{"null byte", "host\x00", false},
	}
	for _, c := range cases {
		if got := validHostnameHint(c.in); got != c.want {
			t.Errorf("%s: validHostnameHint(%q) = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}
