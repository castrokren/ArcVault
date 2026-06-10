package server

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"arcvault/coordinator/internal/bootstrap"
	"arcvault/coordinator/internal/tlscert"
)

// handleBootstrapScript generates and serves the PowerShell bootstrap script.
// Admin-only endpoint. Mints a fresh agent token and embeds it in the script.
func (s *Server) handleBootstrapScript(w http.ResponseWriter, r *http.Request) {
	// Read the coordinator cert PEM
	certPEM, err := tlscert.ReadCertPEM(s.cfg.CertFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read cert: %v", err), http.StatusInternalServerError)
		return
	}

	// Mint a fresh agent token
	agentToken, err := s.db.CreateAgentToken("bootstrap")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to mint agent token: %v", err), http.StatusInternalServerError)
		return
	}

	// Compute cert SHA-1 thumbprint for TLS pinning on PS 5.1
	certHash := sha1.Sum(certPEM)
	certThumbprint := fmt.Sprintf("%X", certHash)

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
		CertThumbprint: certThumbprint,
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

