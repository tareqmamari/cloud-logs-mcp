package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/observability-c/logs-mcp-server/internal/config"
	"github.com/observability-c/logs-mcp-server/internal/server"
	"go.uber.org/zap"
)

const version = "0.1.0"

func main() {
	// Load .env file if it exists (optional, for development)
	_ = godotenv.Load()

	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

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
		zap.String("endpoint", cfg.ServiceURL),
	}
	if cfg.InstanceName != "" {
		logFields = append(logFields, zap.String("instance", cfg.InstanceName))
	}
	logger.Info("Starting IBM Cloud Logs MCP Server", logFields...)

	// Create and start MCP server
	mcpServer, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create MCP server", zap.Error(err))
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start server
	if err := mcpServer.Start(ctx); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}

	logger.Info("Server shutdown complete")
}

func initLogger() (*zap.Logger, error) {
	env := os.Getenv("ENVIRONMENT")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
