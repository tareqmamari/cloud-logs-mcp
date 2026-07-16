package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

// Contract tests verify that ALL registered tools satisfy interface invariants.
// These tests catch regressions when new tools are added or existing ones modified.

func TestContract_AllToolsHaveValidNames(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			if name == "" {
				t.Fatal("Tool name must not be empty")
			}

			// Names should be lowercase with underscores (snake_case)
			for _, r := range name {
				if !unicode.IsLower(r) && r != '_' && !unicode.IsDigit(r) {
					t.Errorf("Tool name %q should be snake_case (contains %q)", name, string(r))
					break
				}
			}

			// Names should not start or end with underscores
			if strings.HasPrefix(name, "_") || strings.HasSuffix(name, "_") {
				t.Errorf("Tool name %q should not start/end with underscore", name)
			}

			// Names should not contain double underscores
			if strings.Contains(name, "__") {
				t.Errorf("Tool name %q should not contain double underscores", name)
			}

			// Reasonable length
			if len(name) > 50 {
				t.Errorf("Tool name %q is too long (%d chars)", name, len(name))
			}
		})
	}
}

func TestContract_AllToolsHaveDescriptions(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			desc := tool.Description()
			if desc == "" {
				t.Errorf("Tool %q has empty description", name)
			}
			if len(desc) < 10 {
				t.Errorf("Tool %q description too short (%d chars): %q", name, len(desc), desc)
			}
			// Description should start with a capital letter or verb
			if len(desc) > 0 && unicode.IsLower(rune(desc[0])) {
				t.Errorf("Tool %q description should start with capital letter: %q", name, desc[:min(50, len(desc))])
			}
		})
	}
}

func TestContract_AllToolsHaveValidInputSchema(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			schema := tool.InputSchema()
			if schema == nil {
				t.Fatalf("Tool %q InputSchema() returned nil", name)
			}

			// Schema should be JSON-serializable
			data, err := json.Marshal(schema)
			if err != nil {
				t.Fatalf("Tool %q InputSchema() not JSON-serializable: %v", name, err)
			}

			// Should be a valid JSON object (not array, string, etc.)
			var schemaMap map[string]interface{}
			if err := json.Unmarshal(data, &schemaMap); err != nil {
				t.Fatalf("Tool %q InputSchema() is not a JSON object: %v", name, err)
			}

			// Should have "type" field set to "object"
			if schemaType, ok := schemaMap["type"]; ok {
				if schemaType != "object" {
					t.Errorf("Tool %q InputSchema() type = %q, want %q", name, schemaType, "object")
				}
			}
		})
	}
}

func TestContract_AllToolsHaveValidAnnotations(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			annotations := tool.Annotations()
			// Annotations can be nil (defaults used), that's OK
			if annotations == nil {
				return
			}

			// If annotations are set, verify they're reasonable
			// Annotations should be JSON-serializable
			_, err := json.Marshal(annotations)
			if err != nil {
				t.Errorf("Tool %q Annotations() not JSON-serializable: %v", name, err)
			}
		})
	}
}

func TestContract_AllToolsHaveNonNegativeTimeout(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			timeout := tool.DefaultTimeout()
			if timeout < 0 {
				t.Errorf("Tool %q DefaultTimeout() = %v, must be >= 0", name, timeout)
			}
		})
	}
}

func TestContract_ExecuteWithNilArgs_NoPanic(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			// Execute with nil arguments should NOT panic
			// It may return an error (expected) but must not crash
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Tool %q panicked with nil args: %v", name, r)
				}
			}()

			ctx := context.Background()
			_, _ = tool.Execute(ctx, nil)
			// We don't check the result - just verifying no panic
		})
	}
}

func TestContract_ExecuteWithEmptyArgs_NoPanic(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Tool %q panicked with empty args: %v", name, r)
				}
			}()

			ctx := context.Background()
			_, _ = tool.Execute(ctx, map[string]interface{}{})
		})
	}
}

func TestContract_ExecuteWithCancelledContext_NoPanic(t *testing.T) {
	tools := GetAllTools(nil, nil)

	for _, tool := range tools {
		name := tool.Name()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Tool %q panicked with cancelled context: %v", name, r)
				}
			}()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			_, _ = tool.Execute(ctx, map[string]interface{}{})
		})
	}
}

func TestContract_UniqueToolNames(t *testing.T) {
	tools := GetAllTools(nil, nil)
	names := make(map[string]int)

	for _, tool := range tools {
		names[tool.Name()]++
	}

	for name, count := range names {
		if count > 1 {
			t.Errorf("Tool name %q appears %d times (must be unique)", name, count)
		}
	}
}

func TestContract_ToolNamingConventions(t *testing.T) {
	tools := GetAllTools(nil, nil)

	// Verify common naming patterns
	crudPrefixes := map[string]bool{
		"get_": true, "list_": true, "create_": true,
		"update_": true, "delete_": true, "replace_": true,
	}

	for _, tool := range tools {
		name := tool.Name()

		// Check CRUD tools have matching descriptions
		for prefix := range crudPrefixes {
			if strings.HasPrefix(name, prefix) {
				desc := strings.ToLower(tool.Description())
				verb := strings.TrimSuffix(prefix, "_")

				// get_ tools should mention "get" or "retrieve" in description
				if verb == "get" && !strings.Contains(desc, "get") && !strings.Contains(desc, "retriev") && !strings.Contains(desc, "fetch") {
					t.Logf("Warning: Tool %q description doesn't mention get/retrieve: %q", name, tool.Description()[:min(80, len(tool.Description()))])
				}
				// delete_ tools should mention "delete" or "remove"
				if verb == "delete" && !strings.Contains(desc, "delet") && !strings.Contains(desc, "remov") {
					t.Logf("Warning: Tool %q description doesn't mention delete/remove: %q", name, tool.Description()[:min(80, len(tool.Description()))])
				}
				break
			}
		}
	}
}

func TestContract_EnhancedToolMetadata(t *testing.T) {
	tools := GetAllTools(nil, nil)

	enhancedCount := 0
	for _, tool := range tools {
		et, ok := tool.(EnhancedTool)
		if !ok {
			continue
		}
		enhancedCount++

		name := tool.Name()
		meta := et.Metadata()

		t.Run(name, func(t *testing.T) {
			if meta == nil {
				t.Errorf("EnhancedTool %q returned nil Metadata()", name)
				return
			}

			if len(meta.Categories) == 0 {
				t.Errorf("EnhancedTool %q has no categories", name)
			}

			if len(meta.Keywords) == 0 {
				t.Errorf("EnhancedTool %q has no keywords", name)
			}

			validComplexity := map[string]bool{
				ComplexitySimple: true, ComplexityModerate: true,
				ComplexityIntermediate: true, ComplexityAdvanced: true,
				"": true, // Empty is allowed
			}
			if !validComplexity[meta.Complexity] {
				t.Errorf("EnhancedTool %q has invalid complexity: %q", name, meta.Complexity)
			}

			validPositions := map[string]bool{
				ChainStart: true, ChainStarter: true, ChainMiddle: true,
				ChainEnd: true, ChainFinisher: true, "": true,
			}
			if !validPositions[meta.ChainPosition] {
				t.Errorf("EnhancedTool %q has invalid chain position: %q", name, meta.ChainPosition)
			}
		})
	}

	t.Logf("Found %d EnhancedTool implementations out of %d total tools", enhancedCount, len(tools))
}

func TestContract_ToolCount_Minimum(t *testing.T) {
	tools := GetAllTools(nil, nil)

	// This serves as a regression guard - if tools are accidentally removed,
	// this test will catch it
	const expectedMinimum = 80
	if len(tools) < expectedMinimum {
		t.Errorf("Expected at least %d tools, got %d. Were tools accidentally removed?",
			expectedMinimum, len(tools))
	}
}

// TestContract_GetAllToolsMatchesDescriptors asserts that the list GetAllTools
// returns is exactly the descriptor table — the same set the server registers
// (see internal/server registerTools and the golden contract in
// internal/server/testdata/registered_tools.golden.json). GetAllTools,
// GetToolCount and the server registration must never disagree.
func TestContract_GetAllToolsMatchesDescriptors(t *testing.T) {
	descriptors := Descriptors()
	all := GetAllTools(nil, nil)

	if len(all) != len(descriptors) {
		t.Fatalf("GetAllTools returned %d tools, descriptor table has %d", len(all), len(descriptors))
	}
	if GetToolCount() != len(descriptors) {
		t.Fatalf("GetToolCount() = %d, want %d (len of descriptor table)", GetToolCount(), len(descriptors))
	}

	// Exact regression guard on the served-set size. If this changes, the
	// server's exposed tool set changed — update the golden contract too.
	const expectedRegisteredTools = 96
	if len(all) != expectedRegisteredTools {
		t.Errorf("registered tool count = %d, want %d. If this is intentional, update the golden contract in internal/server/testdata.",
			len(all), expectedRegisteredTools)
	}

	// The descriptor-constructed names and the GetAllTools names must match.
	descNames := make(map[string]bool, len(descriptors))
	for _, d := range descriptors {
		descNames[d.New(nil, nil).Name()] = true
	}
	for _, tool := range all {
		if !descNames[tool.Name()] {
			t.Errorf("GetAllTools produced tool %q absent from the descriptor table", tool.Name())
		}
	}
}
