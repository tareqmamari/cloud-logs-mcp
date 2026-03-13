package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/cloud-logs-mcp/internal/client"
)

// mockDoer implements client.Doer for testing without network.
type mockDoer struct{}

func (m *mockDoer) Do(_ context.Context, _ *client.Request) (*client.Response, error) {
	return &client.Response{StatusCode: 200, Body: []byte("{}")}, nil
}
func (m *mockDoer) GetInstanceInfo() client.InstanceInfo {
	return client.InstanceInfo{Region: "us-south"}
}
func (m *mockDoer) Close() error { return nil }

// TestMCPWirePayload captures the REAL JSON payload that the MCP server
// sends to clients during tools/list. This measures the actual protocol
// overhead — the exact bytes that enter the agent's context window.
//
// Run: go test -v -run TestMCPWirePayload ./internal/tools/
func TestMCPWirePayload(t *testing.T) {
	logger := zap.NewNop()
	mockClient := &mockDoer{}

	allTools := GetAllTools(mockClient, logger)

	// Build MCP tool definitions exactly as the server does (server.go:registerTool)
	var mcpTools []mcp.Tool
	for _, tool := range allTools {
		mcpTools = append(mcpTools, mcp.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: tool.Annotations(),
		})
	}

	// ═══ 1. Measure the tools/list payload ═══

	toolListJSON, err := json.Marshal(mcpTools)
	if err != nil {
		t.Fatalf("Failed to marshal tools: %v", err)
	}

	// Full JSON-RPC response wrapper
	fullResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  map[string]interface{}{"tools": mcpTools},
	}
	fullResponseJSON, _ := json.Marshal(fullResponse)

	// ═══ 2. Measure per-tool breakdown ═══

	type toolMeasurement struct {
		Name        string `json:"name"`
		TotalBytes  int    `json:"total_bytes"`
		DescBytes   int    `json:"desc_bytes"`
		SchemaBytes int    `json:"schema_bytes"`
		DescText    string `json:"desc_text,omitempty"`
	}
	var measurements []toolMeasurement

	for _, tool := range allTools {
		descJSON, _ := json.Marshal(tool.Description())
		schemaJSON, _ := json.Marshal(tool.InputSchema())
		fullTool := mcp.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
			Annotations: tool.Annotations(),
		}
		toolJSON, _ := json.Marshal(fullTool)

		measurements = append(measurements, toolMeasurement{
			Name:        tool.Name(),
			TotalBytes:  len(toolJSON),
			DescBytes:   len(descJSON),
			SchemaBytes: len(schemaJSON),
		})
	}

	sort.Slice(measurements, func(i, j int) bool {
		return measurements[i].TotalBytes > measurements[j].TotalBytes
	})

	totalDescBytes := 0
	totalSchemaBytes := 0
	for _, m := range measurements {
		totalDescBytes += m.DescBytes
		totalSchemaBytes += m.SchemaBytes
	}

	// ═══ 3. Execute reference tools and measure real responses ═══

	type responseMeasurement struct {
		Name      string `json:"name"`
		TextBytes int    `json:"text_bytes"`
		JSONBytes int    `json:"json_bytes"`
		Error     string `json:"error,omitempty"`
	}
	var responses []responseMeasurement

	// These tools work without a real API client (they return static/computed data)
	refToolArgs := map[string]map[string]interface{}{
		"get_dataprime_reference": {"topic": "commands"},
		"get_query_templates":     {"category": "all"},
		"build_query":             {"description": "errors in production", "time_range": "1h"},
		"validate_query":          {"query": "source logs | filter $m.severity == ERROR"},
		"estimate_query_cost":     {"query": "source logs | filter $m.severity == ERROR"},
		"search_tools":            {"query": "alert"},
		"list_tool_categories":    {},
		"session_context":         {"action": "get"},
	}

	ctx := context.Background()
	for name, args := range refToolArgs {
		var targetTool Tool
		for _, tool := range allTools {
			if tool.Name() == name {
				targetTool = tool
				break
			}
		}
		if targetTool == nil {
			responses = append(responses, responseMeasurement{
				Name: name, Error: "tool not found",
			})
			continue
		}

		result, execErr := targetTool.Execute(ctx, args)
		if execErr != nil {
			responses = append(responses, responseMeasurement{
				Name: name, Error: execErr.Error(),
			})
			continue
		}
		if result == nil {
			responses = append(responses, responseMeasurement{
				Name: name, Error: "nil result",
			})
			continue
		}

		resultJSON, _ := json.Marshal(result)
		textBytes := 0
		for _, content := range result.Content {
			if tc, ok := content.(*mcp.TextContent); ok {
				textBytes += len(tc.Text)
			}
		}
		responses = append(responses, responseMeasurement{
			Name:      name,
			TextBytes: textBytes,
			JSONBytes: len(resultJSON),
		})
	}

	// ═══ 4. Write results ═══

	results := map[string]interface{}{
		"total_tools":                   len(allTools),
		"tools_list_payload_bytes":      len(toolListJSON),
		"full_jsonrpc_response_bytes":   len(fullResponseJSON),
		"total_description_bytes":       totalDescBytes,
		"total_schema_bytes":            totalSchemaBytes,
		"per_tool":                      measurements,
		"reference_tool_response_sizes": responses,
	}

	// Write results to disk only when WRITE_BENCHMARK=1 (avoids file churn during pre-commit)
	if os.Getenv("WRITE_BENCHMARK") == "1" {
		if err := os.MkdirAll("../../benchmarks", 0750); err != nil {
			t.Logf("Warning: could not create benchmarks dir: %v", err)
		}
		resultsJSON, _ := json.MarshalIndent(results, "", "  ")
		resultsJSON = append(resultsJSON, '\n')
		if err := os.WriteFile("../../benchmarks/mcp-wire-payload.json", resultsJSON, 0600); err != nil {
			t.Logf("Warning: could not write results file: %v", err)
		}

		toolListJSON = append(toolListJSON, '\n')
		if err := os.WriteFile("../../benchmarks/mcp-tools-list-raw.json", toolListJSON, 0600); err != nil {
			t.Logf("Warning: could not write raw payload file: %v", err)
		}
	}

	// ═══ 5. Print report ═══

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║         MCP Wire Payload — Real Measured Data               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Total tools registered:       %d\n", len(allTools))
	fmt.Printf("tools/list JSON array:        %s (%d bytes)\n", humanBytes(len(toolListJSON)), len(toolListJSON))
	fmt.Printf("Full JSON-RPC response:       %s (%d bytes)\n", humanBytes(len(fullResponseJSON)), len(fullResponseJSON))
	fmt.Printf("  - Descriptions total:       %s\n", humanBytes(totalDescBytes))
	fmt.Printf("  - Schemas total:            %s\n", humanBytes(totalSchemaBytes))
	fmt.Println()

	fmt.Println("Top 15 largest tool definitions (on the wire):")
	fmt.Println("─────────────────────────────────────────────────────────────")
	for i := 0; i < 15 && i < len(measurements); i++ {
		m := measurements[i]
		fmt.Printf("  %2d. %-35s %6d B  (desc: %d, schema: %d)\n",
			i+1, m.Name, m.TotalBytes, m.DescBytes, m.SchemaBytes)
	}
	fmt.Println()

	fmt.Println("Reference tool real response sizes (no network, executed locally):")
	fmt.Println("─────────────────────────────────────────────────────────────")
	for _, r := range responses {
		if r.Error != "" {
			fmt.Printf("  %-30s  ERROR: %s\n", r.Name, r.Error)
		} else {
			fmt.Printf("  %-30s  %6d B text, %6d B JSON\n", r.Name, r.TextBytes, r.JSONBytes)
		}
	}

	fmt.Println()
	fmt.Println("Results saved to: benchmarks/mcp-wire-payload.json")
}

func humanBytes(b int) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}
