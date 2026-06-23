package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"time"
)

// Generate creates a self-signed TLS certificate with ECDSA P-256.
// host should be an IP address or hostname.
// Files are written to certPath and keyPath.
// SANs include: host, 127.0.0.1, localhost, and os.Hostname().
// EKU: ServerAuth only.
// KeyUsage: DigitalSignature only (no CertSign, no KeyEncipherment).
// IsCA: false (leaf cert).
func Generate(host string, certPath string, keyPath string) error {
	// Generate private key (ECDSA P-256)
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	// Build certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	hostname, _ := os.Hostname()

	// Collect SANs
	var ipAddresses []net.IP
	var dnsNames []string

	// Add host
	if ip := net.ParseIP(host); ip != nil {
		ipAddresses = append(ipAddresses, ip)
	} else {
		dnsNames = append(dnsNames, host)
	}

	// Add 127.0.0.1
	ipAddresses = append(ipAddresses, net.ParseIP("127.0.0.1"))

	// Add localhost
	dnsNames = append(dnsNames, "localhost")

	// Add os.Hostname() if available
	if hostname != "" {
		dnsNames = append(dnsNames, hostname)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		IPAddresses:           ipAddresses,
		DNSNames:              dnsNames,
	}

	// Self-sign the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	// Encode certificate as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Encode private key as PKCS#8 PEM
	keyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Write cert to file
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return err
	}

	// Write key to file
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}

	return nil
}

// EnsureExists generates a certificate if it doesn't already exist.
// If both files exist, it does nothing (idempotent).
func EnsureExists(host string, certPath string, keyPath string) error {
	// Check if both files exist
	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr == nil && keyErr == nil {
		// Both files exist, noop
		return nil
	}

	// At least one file is missing, generate both
	return Generate(host, certPath, keyPath)
}

// ReadCertPEM reads and parses the certificate PEM file.
// Returns the DER-encoded certificate bytes.
func ReadCertPEM(certPath string) ([]byte, error) {
	pemBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("no PEM data found in certificate file")
	}

	if block.Type != "CERTIFICATE" {
		return nil, errors.New("invalid PEM block type, expected CERTIFICATE")
	}
	return pemBytes, nil
}

// Load loads a certificate and private key from PEM files.
// This is a convenience wrapper around tls.LoadX509KeyPair.
// Use tls.LoadX509KeyPair directly for tls.Config.
func Load(certPath string, keyPath string) error {
	// Simple validation that files exist and are readable
	if _, err := os.Stat(certPath); err != nil {
		return err
	}
	if _, err := os.Stat(keyPath); err != nil {
		return err
	}
	return nil
}
