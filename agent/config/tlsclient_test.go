package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// D2: Test BuildTLSConfig returns nil when caCertFile is empty
func TestBuildTLSConfig_emptyReturnsNil(t *testing.T) {
	cfg, err := BuildTLSConfig("")
	if err != nil {
		t.Fatalf("BuildTLSConfig with empty file returned error: %v", err)
	}
	if cfg != nil {
		t.Error("Expected nil config for empty caCertFile, got non-nil")
	}
}

// Helper: generate a simple self-signed cert for testing
func generateTestCert(t *testing.T, commonName string) (certPEM, keyPEM []byte) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames:   []string{"127.0.0.1", "localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:       false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		t.Fatalf("Failed to marshal key: %v", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	return
}

// D2: Test BuildTLSConfig reads and parses PEM correctly
func TestBuildTLSConfig_readsPEM(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.crt")

	// Generate a test cert
	certPEM, _ := generateTestCert(t, "127.0.0.1")
	if err := writeFile(certPath, certPEM); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	cfg, err := BuildTLSConfig(certPath)
	if err != nil {
		t.Fatalf("BuildTLSConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.RootCAs == nil {
		t.Fatal("Expected RootCAs to be set")
	}

	// Verify the pool has exactly 1 cert
	subjects := cfg.RootCAs.Subjects()
	if len(subjects) != 1 {
		t.Errorf("Expected 1 cert in pool, got %d", len(subjects))
	}
}

// D2: Test BuildTLSConfig fails on invalid file
func TestBuildTLSConfig_invalidFile(t *testing.T) {
	_, err := BuildTLSConfig("/nonexistent/path/to/cert.pem")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

// D3: TestAgentTrustsPinnedCert — full E2E test
// Agent with ca_cert_file set verifies successfully;
// without ca_cert_file (or with wrong cert), verification fails.
func TestAgentTrustsPinnedCert(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "coordinator.crt")
	keyPath := filepath.Join(tmpDir, "coordinator.key")

	// Generate test certs
	certPEM, keyPEM := generateTestCert(t, "127.0.0.1")
	if err := writeFile(certPath, certPEM); err != nil {
		t.Fatalf("Failed to write cert: %v", err)
	}
	if err := writeFile(keyPath, keyPEM); err != nil {
		t.Fatalf("Failed to write key: %v", err)
	}

	// Start a TLS server using that cert
	mux := http.NewServeMux()
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(mux)
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("Failed to load cert/key: %v", err)
	}
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
	srv.StartTLS()
	defer srv.Close()

	// Test 1: Agent WITH ca_cert_file set (should succeed)
	tlsConfig, err := BuildTLSConfig(certPath)
	if err != nil {
		t.Fatalf("BuildTLSConfig failed: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	resp, err := client.Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("Request with pinned cert failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test 2: Agent WITHOUT ca_cert_file (empty) should fail on self-signed
	// (because nil config means use system roots, which don't trust self-signed)
	tlsConfig2, _ := BuildTLSConfig("")
	client2 := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig2,
		},
	}

	resp2, err := client2.Get(srv.URL + "/test")
	if err == nil {
		resp2.Body.Close()
		t.Fatal("Expected request WITHOUT ca_cert_file to fail, but it succeeded")
	}
}

// Helper: write file
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
