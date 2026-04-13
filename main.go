// Package main implements the IBM Cloud Logs MCP (Model Context Protocol) server.
//
// This server provides MCP tools for interacting with IBM Cloud Logs service,
// including operations for alerts, policies, queries, log ingestion, and more.
//
// The server supports two transport modes:
//
// 1. Stdio Transport (default): For local MCP clients like Claude Desktop
//    - Requires LOGS_SERVICE_URL and LOGS_API_KEY environment variables
//    - Single-tenant: one user's credentials
//
// 2. HTTP Transport: For multi-tenant deployment (e.g., Code Engine)
//    - Set MCP_TRANSPORT=http and MCP_HTTP_PORT=8080
//    - Credentials passed per-request via HTTP headers
//    - Multi-tenant: each request has different credentials
//
// Configuration environment variables:
//   - MCP_TRANSPORT: "stdio" (default) or "http"
//   - MCP_HTTP_PORT: HTTP port for multi-tenant mode (default: 8080)
//   - LOGS_SERVICE_URL: Service URL (required for stdio mode only)
//   - LOGS_API_KEY: API key (required for stdio mode only)  // pragma: allowlist secret
//   - ENVIRONMENT: Set to "production" for production logging
//
// Example usage (stdio mode):
//
//	export LOGS_SERVICE_URL="https://<instance-id>.api.<region>.logs.cloud.ibm.com"
//	export LOGS_API_KEY="<your-api-key>"
//	./logs-mcp-server
//
// Example usage (HTTP mode for multi-tenant):
//
//	export MCP_TRANSPORT=http
//	export MCP_HTTP_PORT=8080
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

	"github.com/tareqmamari/cloud-logs-mcp/internal/config"
	"github.com/tareqmamari/cloud-logs-mcp/internal/server"
	"github.com/tareqmamari/cloud-logs-mcp/internal/skills"
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
//
// Subcommands:
//
//	logs-mcp-server                    Start the MCP server (default)
//	logs-mcp-server --version          Print version information
//	logs-mcp-server --http             Start in HTTP transport mode (multi-tenant)
//	logs-mcp-server skills install     Install agent skills to ~/.agents/skills/
//	logs-mcp-server skills install --project  Install to ./.agents/skills/ (project-level)
//	logs-mcp-server skills list        List available embedded skills
//	logs-mcp-server skills remove      Remove installed skills
func main() {
	// Handle subcommands before initializing the MCP server
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("logs-mcp-server %s (commit: %s, built by: %s)\n", version, commit, builtBy)
			return
		case "--http":
			// Force HTTP transport mode
			os.Setenv("MCP_TRANSPORT", "http")
		case "skills":
			runSkillsCommand(os.Args[2:])
			return
		}
	}

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

	// Check transport mode
	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio" // default
	}

	if transport == "http" {
		// HTTP transport mode - multi-tenant, credentials per-request
		runHTTPTransport(logger, version)
		return
	}

	// Stdio transport mode - single-tenant, credentials from environment
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
		zap.String("transport", "stdio"),
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

// runHTTPTransport starts the server in HTTP transport mode for multi-tenant deployment.
func runHTTPTransport(logger *zap.Logger, version string) {
	port := 8080
	if portStr := os.Getenv("MCP_HTTP_PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	logger.Info("Starting IBM Cloud Logs MCP Server in HTTP transport mode",
		zap.String("version", version),
		zap.String("transport", "http"),
		zap.Int("port", port),
		zap.String("mode", "multi-tenant"),
	)

	httpServer := server.NewHTTPTransportServer(logger, version, port)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- httpServer.Start(ctx)
	}()

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	case err := <-serverDone:
		if err != nil {
			logger.Error("Server error", zap.Error(err))
		}
		return
	}

	// Wait for graceful shutdown
	select {
	case <-serverDone:
		logger.Info("Server shutdown complete")
	case <-time.After(10 * time.Second):
		logger.Warn("Shutdown timeout exceeded")
	}
}

// runSkillsCommand handles the "skills" subcommand.
func runSkillsCommand(args []string) {
	inst := skills.NewInstaller(SkillsFS, version)

	if len(args) == 0 {
		printSkillsUsage()
		return
	}

	switch args[0] {
	case "install":
		projectLevel := false
		for _, arg := range args[1:] {
			if arg == "--project" || arg == "-p" {
				projectLevel = true
			}
		}

		installed, err := inst.Install("", projectLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		dest := "~/.agents/skills/"
		if projectLevel {
			dest = "./.agents/skills/"
		}
		fmt.Printf("Installed %d skills to %s\n\n", len(installed), dest)
		for _, name := range installed {
			fmt.Printf("  + %s\n", name)
		}
		fmt.Printf("\nSkills are now available to Claude Code, Cursor, Gemini CLI, GitHub Copilot, and other agents.\n")

	case "list":
		skillList, err := inst.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("IBM Cloud Logs Agent Skills (v%s)\n\n", version)
		for _, s := range skillList {
			fmt.Printf("  %-45s %s\n", s.Name, s.Description)
		}
		fmt.Printf("\n%d skills available. Run 'logs-mcp-server skills install' to install.\n", len(skillList))

	case "remove":
		projectLevel := false
		for _, arg := range args[1:] {
			if arg == "--project" || arg == "-p" {
				projectLevel = true
			}
		}

		removed, err := inst.Remove("", projectLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(removed) == 0 {
			fmt.Println("No IBM Cloud Logs skills found to remove.")
		} else {
			fmt.Printf("Removed %d skills:\n\n", len(removed))
			for _, name := range removed {
				fmt.Printf("  - %s\n", name)
			}
		}

	case "help", "--help", "-h":
		printSkillsUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown skills command: %s\n\n", args[0])
		printSkillsUsage()
		os.Exit(1)
	}
}

func printSkillsUsage() {
	fmt.Println(`Usage: logs-mcp-server skills <command> [options]

Manage IBM Cloud Logs agent skills (agentskills.io format).
Skills work with Claude Code, Cursor, Gemini CLI, GitHub Copilot, and 30+ other agents.

Commands:
  install            Install skills to ~/.agents/skills/ (user-level, all projects)
  install --project  Install skills to ./.agents/skills/ (project-level, current project only)
  list               List all available embedded skills
  remove             Remove installed skills from ~/.agents/skills/
  remove --project   Remove installed skills from ./.agents/skills/
  help               Show this help message

Examples:
  logs-mcp-server skills list
  logs-mcp-server skills install
  logs-mcp-server skills install --project
  logs-mcp-server skills remove`)
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
