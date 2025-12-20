//go:build ignore

// This script updates the MCP protocol version in .well-known/agent.json and AGENTS.md.
//
// Usage:
//
//	go run scripts/update-protocol-version.go              # Use default (SDK latest)
//	go run scripts/update-protocol-version.go 2025-11-25   # Specify version
//
// MCP Protocol Versions:
//   - 2025-11-25: Anniversary release with elicitation, tasks, SSE polling
//   - 2025-06-18: Current SDK default (v1.1.0)
//   - 2025-03-26: Streamable HTTP transport
//   - 2024-11-05: Initial stable release
//
// Check SDK support: grep "protocolVersion20" $(go env GOPATH)/pkg/mod/github.com/modelcontextprotocol/go-sdk@*/mcp/shared.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
)

// Supported MCP protocol versions (newest first)
var supportedVersions = []string{
	"2025-11-25", // Anniversary release - elicitation, tasks, SSE polling
	"2025-06-18", // Current Go SDK default
	"2025-03-26", // Streamable HTTP
	"2024-11-05", // Initial stable
}

// DefaultProtocolVersion is the version to use when not specified.
// renovate: datasource=github-tags depName=modelcontextprotocol/specification
const DefaultProtocolVersion = "2025-11-25"

func main() {
	log.SetFlags(0)

	version := DefaultProtocolVersion
	if len(os.Args) > 1 {
		version = os.Args[1]
		if !slices.Contains(supportedVersions, version) {
			log.Fatalf("Unknown protocol version: %s\nSupported: %v", version, supportedVersions)
		}
	}

	// Update agent.json
	if err := updateAgentJSON(version); err != nil {
		log.Fatalf("Failed to update agent.json: %v", err)
	}

	// Update AGENTS.md
	if err := updateAgentsMD(version); err != nil {
		log.Fatalf("Failed to update AGENTS.md: %v", err)
	}

	log.Printf("Updated MCP protocol version to %s", version)
}

func updateAgentJSON(version string) error {
	path := ".well-known/agent.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var agent map[string]interface{}
	if err := json.Unmarshal(data, &agent); err != nil {
		return err
	}

	// Update protocol.version
	if protocol, ok := agent["protocol"].(map[string]interface{}); ok {
		protocol["version"] = version
	}

	output, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(output, '\n'), 0644)
}

func updateAgentsMD(version string) error {
	path := "AGENTS.md"
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	// Update MCP Spec Compliance line
	// Pattern: > **MCP Spec Compliance**: 2025-06-18
	re := regexp.MustCompile(`(\*\*MCP Spec Compliance\*\*:\s*)\d{4}-\d{2}-\d{2}`)
	content = re.ReplaceAllString(content, fmt.Sprintf("${1}%s", version))

	return os.WriteFile(path, []byte(content), 0644)
}
