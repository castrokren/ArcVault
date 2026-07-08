package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// BuildTLSConfig builds a tls.Config for the agent's HTTP client.
// If caCertFile is set, it loads the PEM and sets RootCAs to trust only that cert.
// If caCertFile is empty, it returns nil (use system roots).
func BuildTLSConfig(caCertFile string) (*tls.Config, error) {
	if caCertFile == "" {
		// Empty = use system roots
		return nil, nil
	}

	// Read the PEM file
	pemBytes, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert file: %w", err)
	}

	// Create a cert pool and add the cert
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("failed to parse CA cert PEM")
	}

	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
