package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"arcvault/agent/config"
)

// TestRun_ReportsHTTPStatusOnDialFailure proves that when the coordinator
// refuses the WebSocket upgrade (e.g. an invalid/expired token gets a plain
// 401 instead of a 101 Switching Protocols), the error returned by run()
// includes the actual HTTP status code instead of the bare, undiagnosable
// gorilla/websocket "bad handshake" text. Without this, an operator watching
// the agent's log has no way to tell an auth rejection apart from an origin
// rejection, a proxy error, or anything else that produces a non-101 response.
func TestRun_ReportsHTTPStatusOnDialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	wsURL := buildWSURL(srv.URL, "test-agent", "bad-token")

	c := &Client{
		AgentID:    "test-agent",
		CACertFile: "",
		Tokens:     config.NewTokenStore("bad-token", ""),
	}

	err := c.run(wsURL)
	if err == nil {
		t.Fatal("expected run() to fail against a server that never upgrades, got nil error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("run() error = %q, want it to mention HTTP status 401 so operators can diagnose the rejection", err.Error())
	}
}
