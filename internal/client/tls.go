package client

import (
	"crypto/tls"
)

// newTLSConfig returns a secure TLS configuration with TLS 1.2 minimum.
func newTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}
