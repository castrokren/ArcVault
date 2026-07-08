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
		CertThumbprint: "ABCDEF0123456789",
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
		CertThumbprint: "ABCDEF0123456789",
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
		CertThumbprint: "ABCDEF0123456789",
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
		CertThumbprint: "ABCDEF0123456789",
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
		CertThumbprint: "ABCDEF0123456789",
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
		CertThumbprint: "ABCDEF0123456789",
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

// B5: TestGenerateScript_crossEdition
func TestGenerateScript_crossEdition(t *testing.T) {
	params := Params{
		CoordinatorURL: "https://192.168.1.10",
		AgentToken:     "token123",
		CertPEM:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		CertThumbprint: "ABCDEF0123456789",
		AgentExeSHA256: "0123456789ABCDEF",
	}

	script := GenerateScript(params)

	// Check for PSEdition branching
	if !strings.Contains(script, "PSEdition") {
		t.Error("PSEdition check not found in script")
	}

	// Check for both branches
	if !strings.Contains(script, "-SkipCertificateCheck") {
		t.Error("-SkipCertificateCheck (PS 7+ branch) not found in script")
	}

	if !strings.Contains(script, "ServerCertificateValidationCallback") {
		t.Error("ServerCertificateValidationCallback (PS 5.1 branch) not found in script")
	}
}
