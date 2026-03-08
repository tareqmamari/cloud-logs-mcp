//go:build tlsskipverify

package client

import (
	"crypto/tls"

	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"go.uber.org/zap"
)

// newTLSConfig returns a TLS configuration that optionally disables certificate
// verification. This variant is only compiled with the "tlsskipverify" build tag
// and should never be used in production.
func newTLSConfig(cfg *config.Config, logger *zap.Logger) *tls.Config {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if !cfg.TLSVerify {
		tlsConfig.InsecureSkipVerify = true // #nosec G402 -- guarded by build tag, intentional for testing
		logger.Warn("TLS certificate verification is DISABLED - this is insecure and should only be used for testing",
			zap.String("service_url", cfg.ServiceURL),
		)
	}

	return tlsConfig
}
