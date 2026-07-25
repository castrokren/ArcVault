package bootstrap

import (
	"strings"
	"testing"
)

// B1: TestGenerateScript_containsURL
func TestGenerateScript_containsURL(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	if !strings.Contains(script, "https://192.168.1.10") {
		t.Error("CoordinatorURL not found in script")
	}
}

// B1: TestGenerateScript_containsToken
func TestGenerateScript_containsToken(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "mytoken123abc",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	if !strings.Contains(script, "mytoken123abc") {
		t.Error("AgentToken not found in script")
	}
}

// B1: TestGenerateScript_containsComputername
func TestGenerateScript_containsComputername(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	if !strings.Contains(script, "$env:COMPUTERNAME") {
		t.Error("$env:COMPUTERNAME not found in script")
	}
}

// B2: TestGenerateScript_certInSingleQuotedHeredoc
func TestGenerateScript_certInSingleQuotedHeredoc(t *testing.T) {
	certPEM := "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----"
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        certPEM,
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	// Check that cert is in single-quoted here-string
	if !strings.Contains(script, "@'\n") {
		t.Error("Opening @' not found (expected single-quoted here-string)")
	}

	if !strings.Contains(script, "\n'@") {
		t.Error("Closing '@' not found (expected single-quoted here-string)")
	}

	// Verify the cert is inside the here-string
	idx := strings.Index(script, "@'\n")
	idxEnd := strings.Index(script[idx:], "'@")
	if idxEnd == -1 {
		t.Fatal("Could not find closing '@' after opening @'")
	}

	hereString := script[idx : idx+idxEnd+2]
	if !strings.Contains(hereString, certPEM) {
		t.Error("CertPEM not found inside the single-quoted here-string")
	}
}

// B3: TestGenerateScript_forcesTls12
func TestGenerateScript_forcesTls12(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	if !strings.Contains(script, "Tls12") {
		t.Error("Tls12 not found in script")
	}

	if !strings.Contains(script, "SecurityProtocol") {
		t.Error("SecurityProtocol not found in script")
	}
}

// B4: TestGenerateScript_mandatorySha256
func TestGenerateScript_mandatorySha256(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "AABBCCDD11223344",
	}

	script := GenerateScript(params)

	if !strings.Contains(script, "AABBCCDD11223344") {
		t.Error("AgentExeSHA256 not found in script")
	}

	if !strings.Contains(script, "SHA256") {
		t.Error("SHA256 check not found in script")
	}

	if !strings.Contains(script, "throw") {
		t.Error("Error throwing on hash mismatch not found in script")
	}
}

// B5: TestGenerateScript_pinsTrustToEmbeddedCert
//
// Replaces an older cross-edition test that asserted `PSEdition` branching,
// `-SkipCertificateCheck` and `ServerCertificateValidationCallback`. Those
// describe an abandoned Invoke-WebRequest design — PS 5.1's HttpWebRequest could
// not survive this server's TLS renegotiation, so the script switched to
// curl.exe, which works on both editions and needs no branch. The assertions
// below pin what the script actually relies on for trust.
func TestGenerateScript_pinsTrustToEmbeddedCert(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	// Trust is pinned to the embedded cert, not to the machine's trust store.
	if !strings.Contains(script, "--cacert") {
		t.Error("curl --cacert not found: the download would trust the system store instead of the pinned cert")
	}

	// An error response must not be saved as agent.exe.
	if !strings.Contains(script, "--fail") {
		t.Error("curl --fail not found: an auth/error body could be written as agent.exe")
	}

	// PS 5.1 negotiates SSL3/TLS1.0 by default and cannot reach the coordinator.
	if !strings.Contains(script, "Tls12") {
		t.Error("TLS 1.2 is not forced: PS 5.1 would fail the HTTPS connection")
	}

	// The cert has to be on disk before anything is fetched with it.
	certIdx := strings.Index(script, "Set-Content -Path $CertPath")
	curlIdx := strings.Index(script, "--cacert")
	if certIdx == -1 || curlIdx == -1 || certIdx > curlIdx {
		t.Error("cert must be written before the download that pins to it")
	}

	// Certificate verification must never be bypassed. These are the two ways
	// PowerShell does it, and neither belongs in a script that ships a pinned CA.
	for _, bypass := range []string{"-SkipCertificateCheck", "ServerCertificateValidationCallback", "--insecure"} {
		if strings.Contains(script, bypass) {
			t.Errorf("script disables certificate verification via %q", bypass)
		}
	}
}

// Re-running the script on a machine that already had an agent used to fail:
// curl wrote straight onto agent.exe, Windows locks a running executable, and the
// service was only stopped further down. curl aborted with exit 23
// (CURLE_WRITE_ERROR) before the stop was ever reached. Observed live 2026-07-25
// on SRB3FLPC010: "curl: (23) client returned ERROR on write of 14083 bytes".
func TestGenerateScript_stopsServiceBeforeReplacingBinary(t *testing.T) {
	script := GenerateScript(Params{
		CoordinatorURL: "https://arcvault.lan",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		AgentExeSHA256: "0123456789ABCDEF",
	})

	// The download must target a temp path, never the live binary.
	if !strings.Contains(script, "-o $AgentExeNew") {
		t.Error("download does not go to a temp path; it would overwrite the running agent.exe")
	}

	idx := func(needle string) int {
		i := strings.Index(script, needle)
		if i == -1 {
			t.Fatalf("script is missing %q", needle)
		}
		return i
	}

	download := idx("-o $AgentExeNew")
	hash := idx("Get-FileHash -Path $AgentExeNew")
	stop := idx("sc.exe stop   $ServiceName")
	swap := idx("Move-Item -Path $AgentExeNew")
	install := idx("install-service")

	// Verify before touching anything, stop before swapping, swap before install.
	if !(download < hash && hash < stop && stop < swap && swap < install) {
		t.Errorf("wrong order: download=%d hash=%d stop=%d swap=%d install=%d "+
			"(need download < hash < stop < swap < install)", download, hash, stop, swap, install)
	}

	// A failed download must not leave a working agent broken.
	if !strings.Contains(script, "Remove-Item $AgentExeNew -Force") {
		t.Error("a failed download should clean up the temp file")
	}
}
