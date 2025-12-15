//go:build integration

package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"

	"github.com/tareqmamari/logs-mcp-server/internal/client"
	"github.com/tareqmamari/logs-mcp-server/internal/config"
)

// These tests require real IBM Cloud Logs credentials
// Run with: go test -tags=integration -v ./internal/tools/... -run "TestReal_"

func getTestClient(t *testing.T) *client.Client {
	serviceURL := os.Getenv("LOGS_SERVICE_URL")
	apiKey := os.Getenv("LOGS_API_KEY")
	region := os.Getenv("LOGS_REGION")

	if serviceURL == "" || apiKey == "" {
		t.Skip("Skipping real e2e test: LOGS_SERVICE_URL and LOGS_API_KEY not set")
	}

	if region == "" {
		region = "eu-gb"
	}

	cfg := &config.Config{
		ServiceURL:      serviceURL,
		APIKey:          apiKey,
		Region:          region,
		Timeout:         60 * time.Second,
		MaxRetries:      3,
		RetryWaitMin:    1 * time.Second,
		RetryWaitMax:    30 * time.Second,
		MaxIdleConns:    10,
		IdleConnTimeout: 90 * time.Second,
		TLSVerify:       true,
		EnableRateLimit: false,
	}

	logger, _ := zap.NewDevelopment()
	c, err := client.New(cfg, logger, "test")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return c
}

// TestReal_GlobalModeStrategy tests global mode with real data
func TestReal_GlobalModeStrategy(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	tool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	t.Run("global_system_health", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]interface{}{
			"time_range": "1h",
		})

		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("Tool returned error: %v", result.Content)
		}

		// Extract text content
		text := extractTextContent(result)
		t.Logf("Global mode result:\n%s", text)

		// Verify structure
		if !strings.Contains(text, "# Smart Investigation Report") {
			t.Error("Missing report header")
		}
		if !strings.Contains(text, "**Mode:** global") {
			t.Error("Mode should be global")
		}
		if !strings.Contains(text, "## Root Cause") {
			t.Error("Missing root cause section")
		}
	})
}

// TestReal_ComponentModeStrategy tests component mode with real data
func TestReal_ComponentModeStrategy(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	tool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	// First, let's query to find what applications exist
	t.Run("component_mode_investigation", func(t *testing.T) {
		// Try investigating a common application name pattern
		result, err := tool.Execute(ctx, map[string]interface{}{
			"application": "ibm-logs", // Try a common app
			"time_range":  "6h",
		})

		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("Tool returned error: %v", result.Content)
		}

		text := extractTextContent(result)
		t.Logf("Component mode result:\n%s", text)

		// Verify structure
		if !strings.Contains(text, "**Mode:** component") {
			t.Error("Mode should be component")
		}
		if !strings.Contains(text, "**Target Service:**") {
			t.Error("Should show target service")
		}
	})
}

// TestReal_HeuristicDetection tests heuristic pattern detection with real data
func TestReal_HeuristicDetection(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	tool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	t.Run("detect_real_error_patterns", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]interface{}{
			"time_range": "24h",
		})

		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		text := extractTextContent(result)
		t.Logf("Heuristic detection result:\n%s", text)

		// Check if findings section exists
		if strings.Contains(text, "## Findings") {
			t.Log("Found findings section - heuristics may have detected patterns")
		}

		// Check if suggested actions exist
		if strings.Contains(text, "## Suggested Next Actions") {
			t.Log("Found suggested actions - heuristics are working")
		}
	})
}

// TestReal_RemediationAssetGeneration tests asset generation with real findings
func TestReal_RemediationAssetGeneration(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	tool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	t.Run("generate_assets_from_real_data", func(t *testing.T) {
		result, err := tool.Execute(ctx, map[string]interface{}{
			"time_range":      "6h",
			"generate_assets": true,
		})

		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		text := extractTextContent(result)
		t.Logf("Asset generation result (first 3000 chars):\n%s", truncateForLog(text, 3000))

		// If there are findings, we should see generated assets
		if strings.Contains(text, "## Findings") {
			// Check for Terraform
			if strings.Contains(text, "Terraform") || strings.Contains(text, "resource") {
				t.Log("Terraform configuration generated")
			}
			// Check for dashboard
			if strings.Contains(text, "Dashboard") || strings.Contains(text, "widgets") {
				t.Log("Dashboard configuration generated")
			}
		} else {
			t.Log("No findings detected - no assets generated (this is expected if system is healthy)")
		}
	})
}

// TestReal_QueryExecution tests that queries actually execute against the API
func TestReal_QueryExecution(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	// Create a simple query tool to verify API connectivity
	queryTool := NewQueryTool(c, logger)
	ctx := context.Background()

	t.Run("verify_api_connectivity", func(t *testing.T) {
		now := time.Now().UTC()
		startDate := now.Add(-1 * time.Hour).Format(time.RFC3339)
		endDate := now.Format(time.RFC3339)

		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query":      "source logs | filter $m.severity >= 5 | limit 10",
			"start_date": startDate,
			"end_date":   endDate,
			"tier":       "frequent_search",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if result.IsError {
			text := extractTextContent(result)
			t.Fatalf("Query returned error: %s", text)
		}

		text := extractTextContent(result)
		t.Logf("Query result preview:\n%s", truncateForLog(text, 1000))
		t.Log("API connectivity verified - queries are working")
	})
}

// TestReal_ErrorPatternAnalysis tests the error pattern analysis with real logs
func TestReal_ErrorPatternAnalysis(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	// Use the strategy directly to see raw analysis
	strategy := &GlobalModeStrategy{}

	t.Run("analyze_real_error_patterns", func(t *testing.T) {
		now := time.Now().UTC()
		invCtx := &SmartInvestigationContext{
			Mode: ModeGlobal,
			TimeRange: InvestigationTimeRange{
				Start: now.Add(-1 * time.Hour),
				End:   now,
			},
		}

		// Execute queries using the client
		queryTool := NewQueryTool(c, logger)
		ctx := context.Background()

		// Execute the error rate query
		result, err := queryTool.Execute(ctx, map[string]interface{}{
			"query": `source logs
				| filter $m.severity >= 5
				| groupby $l.applicationname
				| calculate count() as error_count
				| sortby -error_count
				| limit 20`,
			"start_date": invCtx.TimeRange.Start.Format(time.RFC3339),
			"end_date":   invCtx.TimeRange.End.Format(time.RFC3339),
			"tier":       "frequent_search",
		})

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		text := extractTextContent(result)
		t.Logf("Error rate by application:\n%s", truncateForLog(text, 2000))

		// Now test timeline analysis
		result2, err := queryTool.Execute(ctx, map[string]interface{}{
			"query": `source logs
				| filter $m.severity >= 5
				| groupby bucket($m.timestamp, 5m) as time_bucket
				| calculate count() as errors
				| sortby time_bucket`,
			"start_date": invCtx.TimeRange.Start.Format(time.RFC3339),
			"end_date":   invCtx.TimeRange.End.Format(time.RFC3339),
			"tier":       "frequent_search",
		})

		if err != nil {
			t.Fatalf("Timeline query failed: %v", err)
		}

		text2 := extractTextContent(result2)
		t.Logf("Error timeline:\n%s", truncateForLog(text2, 2000))

		// Test evidence synthesis
		evidence := strategy.SynthesizeEvidence(invCtx)
		t.Logf("Synthesized root cause: %s (confidence: %.0f%%)",
			evidence.RootCause, evidence.Confidence*100)
	})
}

// TestReal_FullInvestigationWorkflow tests the complete investigation workflow
func TestReal_FullInvestigationWorkflow(t *testing.T) {
	c := getTestClient(t)
	logger, _ := zap.NewDevelopment()

	tool := NewSmartInvestigateTool(c, logger)
	ctx := context.Background()

	t.Run("complete_workflow", func(t *testing.T) {
		// Step 1: Global scan
		t.Log("Step 1: Running global system health scan...")
		globalResult, err := tool.Execute(ctx, map[string]interface{}{
			"time_range": "1h",
		})
		if err != nil {
			t.Fatalf("Global scan failed: %v", err)
		}

		globalText := extractTextContent(globalResult)
		t.Logf("Global scan findings:\n%s", truncateForLog(globalText, 2000))

		// Step 2: If we found affected services, drill down into one
		if strings.Contains(globalText, "Affected Services:") {
			t.Log("Step 2: Found affected services, drilling down...")

			// Extract first service name (simple parsing)
			lines := strings.Split(globalText, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Affected Services:") {
					// Parse the service name
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						services := strings.TrimSpace(parts[1])
						firstService := strings.Split(services, ",")[0]
						firstService = strings.TrimSpace(firstService)

						if firstService != "" {
							t.Logf("Drilling into service: %s", firstService)

							componentResult, err := tool.Execute(ctx, map[string]interface{}{
								"application": firstService,
								"time_range":  "1h",
							})
							if err != nil {
								t.Logf("Component drill-down failed: %v", err)
							} else {
								componentText := extractTextContent(componentResult)
								t.Logf("Component analysis:\n%s", truncateForLog(componentText, 1500))
							}
						}
					}
					break
				}
			}
		}

		// Step 3: Generate remediation assets
		t.Log("Step 3: Generating remediation assets...")
		assetResult, err := tool.Execute(ctx, map[string]interface{}{
			"time_range":      "1h",
			"generate_assets": true,
		})
		if err != nil {
			t.Fatalf("Asset generation failed: %v", err)
		}

		assetText := extractTextContent(assetResult)

		// Check what assets were generated
		hasAlert := strings.Contains(assetText, "Alert Configuration")
		hasDashboard := strings.Contains(assetText, "Dashboard Configuration")
		hasSOP := strings.Contains(assetText, "Standard Operating Procedures") ||
			strings.Contains(assetText, "Recommended Procedures")

		t.Logf("Assets generated - Alert: %v, Dashboard: %v, SOP: %v",
			hasAlert, hasDashboard, hasSOP)

		if hasAlert || hasDashboard || hasSOP {
			t.Logf("Generated assets:\n%s", truncateForLog(assetText, 3000))
		}
	})
}

// Helper functions

func extractTextContent(result interface{}) string {
	// Handle *mcp.CallToolResult directly
	if mcpResult, ok := result.(*mcp.CallToolResult); ok && mcpResult != nil {
		if len(mcpResult.Content) > 0 {
			if tc, ok := mcpResult.Content[0].(*mcp.TextContent); ok {
				return tc.Text
			}
		}
	}

	// Try direct MCP result type
	if mcpResult, ok := result.(interface{ GetText() string }); ok {
		return mcpResult.GetText()
	}

	return ""
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
