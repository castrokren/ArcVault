package server

import (
	"arcvault/coordinator/internal/bootstrap"
	"arcvault/coordinator/internal/tlscert"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// hostnameHintPattern matches DNS-label characters only. The hint is not
// interpolated into the generated script, but it is persisted as the token's
// agent_id, so it stays bounded and free of separator characters.
var hostnameHintPattern = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

// isLoopbackHost reports whether a coordinator URL points at this machine only.
// Such a script installs fine and then silently fails to reach anything.
func isLoopbackHost(rawURL string) bool {
	host := strings.TrimPrefix(rawURL, "https://")
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]") // IPv6 literal
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

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

// coordinatorBaseURL returns the https base URL an enrolled machine should call
// back on.
//
// `host` is optional in config.json and the installer never writes it, so building
// this as fmt.Sprintf("https://%s", cfg.Host) yielded the literal string
// "https://" — every curl in the generated script then had nowhere to go and the
// machine silently never registered. Falls back to the Host header, which is by
// definition an address that reached us (and already carries a non-default port).
//
// A loopback result is rejected rather than returned: it is correct for the browser
// that asked and useless on the machine being enrolled, and a script that installs
// then never connects is indistinguishable from a broken agent.
func coordinatorBaseURL(cfgHost string, cfgPort int, requestHost string) (string, error) {
	var url string
	switch {
	case cfgHost != "" && cfgPort == 443:
		url = "https://" + cfgHost
	case cfgHost != "":
		url = fmt.Sprintf("https://%s:%d", cfgHost, cfgPort)
	case requestHost != "":
		url = "https://" + requestHost
	default:
		return "", fmt.Errorf("cannot determine this coordinator's address: set \"host\" in config.json to its LAN address or DNS name, then restart the coordinator")
	}

	if isLoopbackHost(url) {
		return "", fmt.Errorf("refusing to generate a script pointing at %s: the machine being enrolled cannot reach that address. Set \"host\" in config.json to this server's LAN address or DNS name, restart the coordinator, and download the script again", url)
	}
	return url, nil
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

	coordinatorURL, err := coordinatorBaseURL(s.cfg.Host, s.cfg.Port, r.Host)
	if err != nil {
		// 409, not 500: the coordinator is fine, its configuration cannot produce a
		// script the target machine could use.
		http.Error(w, err.Error(), http.StatusConflict)
		return
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
