package tlscert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// parseCertFile reads a cert file and returns the parsed certificates.
//
// ReadCertPEM returns PEM, not DER — these tests used to hand its output
// straight to x509.ParseCertificates, which wants DER, and failed with
// "x509: malformed certificate" against perfectly valid certificates. The
// pem.Decode step is the part that was missing.
func parseCertFile(t *testing.T, certPath string) []*x509.Certificate {
	t.Helper()

	pemBytes, err := ReadCertPEM(certPath)
	if err != nil {
		t.Fatalf("ReadCertPEM failed: %v", err)
	}

	block, rest := pem.Decode(pemBytes)
	if block == nil {
		t.Fatalf("no PEM block in %s", certPath)
	}
	if len(rest) != 0 {
		t.Errorf("unexpected trailing data after the certificate PEM block (%d bytes)", len(rest))
	}

	certs, err := x509.ParseCertificates(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	if len(certs) == 0 {
		t.Fatal("no certificates in PEM block")
	}
	return certs
}

// A1: TestGenerate_filesAndParse
func TestGenerate_filesAndParse(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	err := Generate("192.168.1.10", certPath, keyPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Both files should exist
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Fatal("cert.pem not created")
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("key.pem not created")
	}

	// Cert should parse
	parseCertFile(t, certPath)
}

// A2: TestGenerate_SANs
func TestGenerate_SANs(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	host := "192.168.1.10"

	err := Generate(host, certPath, keyPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	certs := parseCertFile(t, certPath)

	cert := certs[0]

	// Check IPAddresses
	ips := make(map[string]bool)
	for _, ip := range cert.IPAddresses {
		ips[ip.String()] = true
	}

	if !ips["127.0.0.1"] {
		t.Error("127.0.0.1 not in IPAddresses")
	}
	if !ips[host] {
		t.Errorf("%s not in IPAddresses", host)
	}

	// Check DNSNames
	names := make(map[string]bool)
	for _, name := range cert.DNSNames {
		names[name] = true
	}

	if !names["localhost"] {
		t.Error("localhost not in DNSNames")
	}

	hostname, _ := os.Hostname()
	if hostname != "" && !names[hostname] {
		t.Errorf("os.Hostname() %s not in DNSNames", hostname)
	}
}

// A3: TestGenerate_leafEKU
func TestGenerate_leafEKU(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	err := Generate("192.168.1.10", certPath, keyPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	certs := parseCertFile(t, certPath)

	cert := certs[0]

	// Check IsCA
	if cert.IsCA {
		t.Error("IsCA should be false for leaf cert")
	}

	// Check ExtKeyUsage
	hasServerAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("ExtKeyUsageServerAuth not set")
	}

	// Check KeyUsage (should be DigitalSignature only)
	expectedKeyUsage := x509.KeyUsageDigitalSignature
	if cert.KeyUsage != expectedKeyUsage {
		t.Errorf("KeyUsage = %d, want %d (DigitalSignature only)", cert.KeyUsage, expectedKeyUsage)
	}
}

// A4: TestTLSHandshake
func TestTLSHandshake(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	err := Generate("127.0.0.1", certPath, keyPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Create an unstarted TLS server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(mux)

	// Load the cert and key
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadX509KeyPair failed: %v", err)
	}

	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	srv.StartTLS()
	defer srv.Close()

	// Create a client with the generated cert in its CA pool
	certs := parseCertFile(t, certPath)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(certs[0])

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// Make a request — should succeed
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// A5: TestTLSHandshake_wrongHost
func TestTLSHandshake_wrongHost(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// Generate cert for 192.168.1.10
	err := Generate("192.168.1.10", certPath, keyPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Create an unstarted TLS server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewUnstartedServer(mux)

	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadX509KeyPair failed: %v", err)
	}

	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}

	srv.StartTLS()
	defer srv.Close()

	// Create a client with the generated cert
	certs := parseCertFile(t, certPath)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(certs[0])

	// Dial with explicit ServerName that is NOT in the cert's SANs
	// The cert has 192.168.1.10, 127.0.0.1, localhost, and os.Hostname()
	// We'll try to verify as "wronghost.invalid" which won't be in there
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		ServerName: "wronghost.invalid",
	}

	dialer := &net.Dialer{}
	conn, err := dialer.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("TCP dial failed: %v", err)
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	tlsConn.Close()

	if err == nil {
		t.Fatal("Expected TLS handshake to fail due to ServerName mismatch, but it succeeded")
	}
	// Should be a certificate verification error
}

// A6: TestEnsureExists_createsAndNoops
func TestEnsureExists_createsAndNoops(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// First call should create files
	err := EnsureExists("192.168.1.10", certPath, keyPath)
	if err != nil {
		t.Fatalf("First EnsureExists failed: %v", err)
	}

	stat1Cert, err := os.Stat(certPath)
	if err != nil {
		t.Fatal("cert.pem not created")
	}

	stat1Key, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal("key.pem not created")
	}

	// Sleep a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Second call should noop (not regenerate)
	err = EnsureExists("192.168.1.10", certPath, keyPath)
	if err != nil {
		t.Fatalf("Second EnsureExists failed: %v", err)
	}

	stat2Cert, err := os.Stat(certPath)
	if err != nil {
		t.Fatal("cert.pem disappeared")
	}

	stat2Key, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal("key.pem disappeared")
	}

	// Mtimes should be unchanged
	if stat1Cert.ModTime() != stat2Cert.ModTime() {
		t.Error("cert.pem was regenerated (mtime changed)")
	}

	if stat1Key.ModTime() != stat2Key.ModTime() {
		t.Error("key.pem was regenerated (mtime changed)")
	}
}
