//go:build !tlsskipverify

package client

import (
	"crypto/tls"

	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"go.uber.org/zap"
)

// newTLSConfig returns a secure TLS configuration.
// TLS certificate verification is always enabled in production builds.
func newTLSConfig(cfg *config.Config, logger *zap.Logger) *tls.Config {
	if !cfg.TLSVerify {
		logger.Warn("TLS certificate verification cannot be disabled in production builds - ignoring tls_verify=false",
			zap.String("service_url", cfg.ServiceURL),
		)
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}
