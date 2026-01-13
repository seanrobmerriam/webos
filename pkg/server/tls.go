package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSConfig holds TLS configuration for the server.
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// LoadCertificates loads TLS certificates from files.
func (c *TLSConfig) LoadCertificates() ([]tls.Certificate, error) {
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("certfile and keyfile must be specified")
	}

	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	return []tls.Certificate{cert}, nil
}

// ServerTLSConfig returns a secure TLS configuration for the server.
// It enables TLS 1.3 and uses strong cipher suites.
func ServerTLSConfig(certificates []tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: certificates,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		NextProtos:               []string{"h2", "http/1.1"},
		PreferServerCipherSuites: true,
	}
}

// ClientTLSConfig creates a TLS configuration for client connections.
// If CAFile is provided, it will be used to verify the server certificate.
func ClientTLSConfig(caFile string) (*tls.Config, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		config.RootCAs = caPool
	}

	return config, nil
}
