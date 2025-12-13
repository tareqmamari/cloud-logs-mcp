// Package main implements the IBM Cloud Logs MCP (Model Context Protocol) server.
//
// This server provides MCP tools for interacting with IBM Cloud Logs service,
// including operations for alerts, policies, queries, log ingestion, and more.
//
// The server communicates using the MCP protocol over stdio, making it compatible
// with Claude Desktop and other MCP clients.
//
// Configuration is provided through environment variables:
//   - LOGS_SERVICE_URL: The IBM Cloud Logs service endpoint URL (required)
//   - LOGS_API_KEY: IBM Cloud API key for authentication (required)  // pragma: allowlist secret
//   - LOGS_REGION: IBM Cloud region (optional - auto-extracted from service URL)
//   - LOGS_INSTANCE_NAME: (Optional) Friendly name for the instance
//   - ENVIRONMENT: (Optional) Set to "production" for production logging
//
// Example usage:
//
//	export LOGS_SERVICE_URL="https://<instance-id>.api.<region>.logs.cloud.ibm.com"
//	export LOGS_API_KEY="<your-api-key>"
//	./logs-mcp-server
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/config"
	"github.com/tareqmamari/logs-mcp-server/internal/server"
)

// Build information - set at build time via ldflags
// For GoReleaser builds: -X main.version={{.Version}} -X main.commit={{.Commit}} ...
// For manual builds: make build VERSION=0.5.0
var (
	version = "dev"     // e.g., "v0.4.0" or "dev"
	commit  = "unknown" // Git commit SHA
	builtBy = "manual"  // "goreleaser" or "manual"
)

// main is the entry point for the IBM Cloud Logs MCP server.
// It initializes the server, loads configuration, and handles graceful shutdown.
func main() {
	// Load .env file if it exists (optional, for development)
	_ = godotenv.Load()

	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync() // Ignore error on cleanup
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	logFields := []zap.Field{
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("built_by", builtBy),
		zap.String("endpoint", cfg.ServiceURL),
	}
	if cfg.InstanceName != "" {
		logFields = append(logFields, zap.String("instance", cfg.InstanceName))
	}
	logger.Info("Starting IBM Cloud Logs MCP Server", logFields...)

	// Create and start MCP server
	mcpServer, err := server.New(cfg, logger, version)
	if err != nil {
		logger.Fatal("Failed to create MCP server", zap.Error(err))
	}

	// Setup graceful shutdown with timeout
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to signal server completion
	serverDone := make(chan error, 1)

	go func() {
		serverDone <- mcpServer.Start(ctx)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case err := <-serverDone:
		if err != nil {
			logger.Error("Server error", zap.Error(err))
		}
		cancel()
		return
	}

	// Initiate graceful shutdown with timeout
	logger.Info("Initiating graceful shutdown", zap.Duration("timeout", cfg.ShutdownTimeout))
	cancel()

	// Wait for server to finish with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	select {
	case <-serverDone:
		logger.Info("Server shutdown complete")
	case <-shutdownCtx.Done():
		logger.Warn("Shutdown timeout exceeded, forcing exit",
			zap.Duration("timeout", cfg.ShutdownTimeout))
	}

	// Allow a brief moment for final cleanup
	time.Sleep(100 * time.Millisecond)
}

// initLogger initializes and returns a zap logger.
// It creates a production logger if ENVIRONMENT=production, otherwise returns
// a development logger with more verbose output.
func initLogger() (*zap.Logger, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
