package server

import (
	"arcvault/coordinator/internal/bootstrap"
	"arcvault/coordinator/internal/tlscert"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// hostnameHintPattern matches DNS-label characters only. The hint is not
// interpolated into the generated script, but it is persisted as the token's
// agent_id, so it stays bounded and free of separator characters.
var hostnameHintPattern = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

// validHostnameHint reports whether the ?hostname= query value is safe to tag a
// bootstrap token with. Empty is valid — the hint is optional.
func validHostnameHint(h string) bool {
	if h == "" {
		return true
	}
	if len(h) > 253 {
		return false
	}
	return hostnameHintPattern.MatchString(h)
}

// handleBootstrapScript generates and serves the PowerShell bootstrap script.
// Admin-only endpoint. Mints a fresh agent token and embeds it in the script.
func (s *Server) handleBootstrapScript(w http.ResponseWriter, r *http.Request) {
	// Optional: caller passes ?hostname=WORKSTATION01 to get a per-machine token.
	// Falls back to "bootstrap" role tag if not provided.
	hostnameHint := r.URL.Query().Get("hostname")
	if !validHostnameHint(hostnameHint) {
		http.Error(w, "invalid hostname: letters, digits, dot and hyphen only, max 253 chars", http.StatusBadRequest)
		return
	}
	tokenRole := "bootstrap"
	if hostnameHint != "" {
		tokenRole = "bootstrap:" + hostnameHint
	}

	// Read the coordinator cert PEM
	certPEM, err := tlscert.ReadCertPEM(s.cfg.CertFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read cert: %v", err), http.StatusInternalServerError)
		return
	}

	// Mint a fresh agent token
	agentToken, err := s.db.CreateAgentToken(tokenRole)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to mint agent token: %v", err), http.StatusInternalServerError)
		return
	}

	// Compute agent.exe SHA-256 for integrity check
	exePath, err := os.Executable()
	if err != nil {
		http.Error(w, "failed to get executable path", http.StatusInternalServerError)
		return
	}

	agentExePath := filepath.Join(filepath.Dir(exePath), "agent.exe")
	agentExeData, err := os.ReadFile(agentExePath)
	if err != nil {
		http.Error(w, "agent.exe not found in coordinator directory", http.StatusInternalServerError)
		return
	}

	agentExeSHA256 := fmt.Sprintf("%X", sha256.Sum256(agentExeData))

	// Build coordinator URL (omit port if 443)
	coordinatorURL := fmt.Sprintf("https://%s", s.cfg.Host)
	if s.cfg.Port != 443 {
		coordinatorURL = fmt.Sprintf("https://%s:%d", s.cfg.Host, s.cfg.Port)
	}

	// Generate the script
	params := bootstrap.Params{
		CoordinatorURL: coordinatorURL,
		AgentToken:     agentToken,
		CertPEM:        string(certPEM),
		AgentExeSHA256: agentExeSHA256,
	}

	script := bootstrap.GenerateScript(params)

	// Serve as attachment
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=bootstrap.ps1")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(script)))
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, script)
}
