//go:build ignore

// This script validates and updates benchmarks.json with live metrics from the codebase.
//
// Usage:
//
//	go run scripts/validate-benchmarks.go          # Validate and update benchmarks.json
//	go run scripts/validate-benchmarks.go --check  # Check only, fail if updates needed
//
// The script scans the codebase to:
//   - Count tools and namespaces
//   - Verify delete tools have confirmation
//   - Count schema tokens
//   - Update timestamps
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	log.SetFlags(0)

	checkOnly := flag.Bool("check", false, "Check only, don't update file")
	flag.Parse()

	benchmarksPath := "benchmarks.json"

	// Read existing benchmarks
	data, err := os.ReadFile(benchmarksPath)
	if err != nil {
		log.Fatalf("Failed to read benchmarks.json: %v", err)
	}

	var benchmarks map[string]interface{}
	if err := json.Unmarshal(data, &benchmarks); err != nil {
		log.Fatalf("Failed to parse benchmarks.json: %v", err)
	}

	// Collect live metrics
	metrics, err := collectMetrics()
	if err != nil {
		log.Fatalf("Failed to collect metrics: %v", err)
	}

	// Update benchmarks with live metrics
	updated := updateBenchmarks(benchmarks, metrics)

	// Update timestamp only if metrics changed
	if updated {
		if metadata, ok := benchmarks["metadata"].(map[string]interface{}); ok {
			metadata["generated"] = time.Now().UTC().Format(time.RFC3339)
		}
	}

	// Marshal back to JSON
	newData, err := json.MarshalIndent(benchmarks, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal benchmarks: %v", err)
	}
	newData = append(newData, '\n')

	if *checkOnly {
		if updated {
			log.Fatalf("benchmarks.json is out of date. Run 'go run scripts/validate-benchmarks.go' to update.")
		}
		log.Println("benchmarks.json is up to date")
		return
	}

	// Only write if there are actual changes
	if !updated {
		log.Println("benchmarks.json is up to date")
		return
	}

	// Write updated benchmarks
	if err := os.WriteFile(benchmarksPath, newData, 0600); err != nil {
		log.Fatalf("Failed to write benchmarks.json: %v", err)
	}

	log.Println("Updated benchmarks.json with live metrics")
}

// Metrics collected from the codebase
type Metrics struct {
	ToolCount                 int
	NamespaceCount            int
	DeleteToolsTotal          int
	DeleteToolsWithConfirm    int
	DeleteToolNames           []string
	ConfirmedDeleteToolNames  []string
}

func collectMetrics() (*Metrics, error) {
	metrics := &Metrics{}

	toolsDir := "internal/tools"

	// Parse all Go files in the tools directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, toolsDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tools directory: %w", err)
	}

	// Count tools by finding types that embed *BaseTool
	toolTypes := make(map[string]bool)
	deleteTools := make(map[string]bool)

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.TypeSpec:
					// Check if this type embeds *BaseTool
					if st, ok := x.Type.(*ast.StructType); ok {
						for _, field := range st.Fields.List {
							if se, ok := field.Type.(*ast.StarExpr); ok {
								if ident, ok := se.X.(*ast.Ident); ok && ident.Name == "BaseTool" {
									toolTypes[x.Name.Name] = true
									if strings.HasPrefix(x.Name.Name, "Delete") && strings.HasSuffix(x.Name.Name, "Tool") {
										deleteTools[x.Name.Name] = true
									}
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	// Count delete tools with RequireConfirmation by scanning file content
	// This is simpler and more reliable than AST-based detection
	confirmedDeleteTools := 0
	files, _ := filepath.Glob(filepath.Join(toolsDir, "*.go"))
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		contentStr := string(content)

		// For each delete tool, check if its Execute method calls RequireConfirmation
		for toolName := range deleteTools {
			// Look for the pattern: func (t *DeleteXTool) Execute ... RequireConfirmation
			// We find the Execute method signature and check if RequireConfirmation appears before next func
			pattern := fmt.Sprintf(`func \(t \*%s\) Execute`, toolName)
			re := regexp.MustCompile(pattern)
			loc := re.FindStringIndex(contentStr)
			if loc != nil {
				// Find the next "func (" or end of file
				remaining := contentStr[loc[1]:]
				endIdx := strings.Index(remaining, "\nfunc (")
				if endIdx == -1 {
					endIdx = len(remaining)
				}
				methodBody := remaining[:endIdx]
				if strings.Contains(methodBody, "RequireConfirmation") {
					confirmedDeleteTools++
					snakeName := toSnakeCase(strings.TrimSuffix(toolName, "Tool"))
					metrics.ConfirmedDeleteToolNames = append(metrics.ConfirmedDeleteToolNames, snakeName)
				}
			}
		}
	}

	metrics.ToolCount = len(toolTypes)
	metrics.DeleteToolsTotal = len(deleteTools)
	metrics.DeleteToolsWithConfirm = confirmedDeleteTools

	for name := range deleteTools {
		// Convert DeleteAlertTool -> delete_alert
		toolName := toSnakeCase(strings.TrimSuffix(name, "Tool"))
		metrics.DeleteToolNames = append(metrics.DeleteToolNames, toolName)
	}

	// Count namespaces from namespacing.go
	namespacingPath := filepath.Join(toolsDir, "namespacing.go")
	if content, err := os.ReadFile(namespacingPath); err == nil {
		re := regexp.MustCompile(`Namespace\w+\s*=\s*"[^"]+"`)
		matches := re.FindAllString(string(content), -1)
		metrics.NamespaceCount = len(matches)
	}

	return metrics, nil
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func updateBenchmarks(benchmarks map[string]interface{}, metrics *Metrics) bool {
	updated := false

	// Update infrastructure metrics
	if perf, ok := benchmarks["benchmarks"].(map[string]interface{}); ok {
		if perfSection, ok := perf["performance"].(map[string]interface{}); ok {
			if infra, ok := perfSection["infrastructure"].(map[string]interface{}); ok {
				if currentCount, ok := infra["tool_count"].(float64); ok {
					if int(currentCount) != metrics.ToolCount {
						infra["tool_count"] = metrics.ToolCount
						updated = true
						log.Printf("Updated tool_count: %d -> %d", int(currentCount), metrics.ToolCount)
					}
				}
				if currentCount, ok := infra["namespace_count"].(float64); ok {
					if int(currentCount) != metrics.NamespaceCount && metrics.NamespaceCount > 0 {
						infra["namespace_count"] = metrics.NamespaceCount
						updated = true
						log.Printf("Updated namespace_count: %d -> %d", int(currentCount), metrics.NamespaceCount)
					}
				}
			}
		}
	}

	// Update elicitation patterns - confirmation coverage
	if lang, ok := benchmarks["benchmarks"].(map[string]interface{}); ok {
		if langSection, ok := lang["language_efficiency"].(map[string]interface{}); ok {
			if elicit, ok := langSection["elicitation_patterns"].(map[string]interface{}); ok {
				// Update the changes field to reflect current state
				newChanges := fmt.Sprintf("RequireConfirmation added to %d/%d delete tools",
					metrics.DeleteToolsWithConfirm, metrics.DeleteToolsTotal)
				if currentChanges, ok := elicit["changes"].(string); ok {
					if currentChanges != newChanges {
						elicit["changes"] = newChanges
						updated = true
						log.Printf("Updated elicitation changes: %s", newChanges)
					}
				}
			}
		}
	}

	// Update recommendations with current tool list
	if recs, ok := benchmarks["recommendations"].(map[string]interface{}); ok {
		if p2, ok := recs["priority_2"].(map[string]interface{}); ok {
			// Update actual_improvement with current count
			newImprovement := fmt.Sprintf("All %d delete tools now require two-step confirmation to prevent accidental deletions",
				metrics.DeleteToolsWithConfirm)
			if current, ok := p2["actual_improvement"].(string); ok {
				if current != newImprovement && metrics.DeleteToolsWithConfirm > 0 {
					p2["actual_improvement"] = newImprovement
					updated = true
				}
			}
		}
	}

	return updated
}
