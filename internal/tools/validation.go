package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidationResult represents the result of a dry-run validation
type ValidationResult struct {
	Valid       bool                   `json:"valid"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// ValidateRequiredFields checks if all required fields are present in the configuration
func ValidateRequiredFields(config map[string]interface{}, requiredFields []string) []string {
	var errors []string
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			errors = append(errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}
	return errors
}

// ValidateEnumField validates that a field value is one of the allowed values
func ValidateEnumField(config map[string]interface{}, field string, allowedValues []string) string {
	val, ok := config[field]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Sprintf("Field '%s' must be a string", field)
	}
	for _, allowed := range allowedValues {
		if strVal == allowed {
			return ""
		}
	}
	return fmt.Sprintf("Invalid value for '%s': got '%s', must be one of: %v", field, strVal, allowedValues)
}

// ValidateStringLength validates that a string field is within length bounds
func ValidateStringLength(config map[string]interface{}, field string, minLen, maxLen int) string {
	val, ok := config[field]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Sprintf("Field '%s' must be a string", field)
	}
	if len(strVal) < minLen {
		return fmt.Sprintf("Field '%s' must be at least %d characters", field, minLen)
	}
	if maxLen > 0 && len(strVal) > maxLen {
		return fmt.Sprintf("Field '%s' must be at most %d characters", field, maxLen)
	}
	return ""
}

// ValidateIntRange validates that an integer field is within range
func ValidateIntRange(config map[string]interface{}, field string, min, max int) string {
	val, ok := config[field]
	if !ok {
		return ""
	}
	var intVal int
	switch v := val.(type) {
	case float64:
		intVal = int(v)
	case int:
		intVal = v
	case int64:
		intVal = int(v)
	default:
		return fmt.Sprintf("Field '%s' must be a number", field)
	}
	if intVal < min {
		return fmt.Sprintf("Field '%s' must be at least %d", field, min)
	}
	if max > 0 && intVal > max {
		return fmt.Sprintf("Field '%s' must be at most %d", field, max)
	}
	return ""
}

// toTitleCase converts a snake_case or camelCase string to Title Case
func toTitleCase(s string) string {
	// Replace underscores with spaces
	s = strings.ReplaceAll(s, "_", " ")
	// Split on spaces
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// FormatDryRunResult creates a formatted response for dry-run validation
func FormatDryRunResult(result *ValidationResult, resourceType string, config map[string]interface{}) *mcp.CallToolResult {
	var builder strings.Builder

	builder.WriteString("## Dry-Run Validation Result\n\n")
	builder.WriteString(fmt.Sprintf("**Resource Type:** %s\n\n", resourceType))

	if result.Valid {
		builder.WriteString("✅ **Status:** Valid - configuration is ready for creation\n\n")
	} else {
		builder.WriteString("❌ **Status:** Invalid - please fix errors before creating\n\n")
	}

	if len(result.Errors) > 0 {
		builder.WriteString("### Errors\n\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("- ❌ %s\n", err))
		}
		builder.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		builder.WriteString("### Warnings\n\n")
		for _, warn := range result.Warnings {
			builder.WriteString(fmt.Sprintf("- ⚠️ %s\n", warn))
		}
		builder.WriteString("\n")
	}

	if len(result.Summary) > 0 {
		builder.WriteString("### Configuration Summary\n\n")
		for key, val := range result.Summary {
			builder.WriteString(fmt.Sprintf("- **%s:** %v\n", toTitleCase(key), val))
		}
		builder.WriteString("\n")
	}

	if len(result.Suggestions) > 0 {
		builder.WriteString("### Suggestions\n\n")
		for _, sug := range result.Suggestions {
			builder.WriteString(fmt.Sprintf("- 💡 %s\n", sug))
		}
		builder.WriteString("\n")
	}

	// Add the raw config for reference
	builder.WriteString("### Submitted Configuration\n\n```json\n")
	configBytes, _ := json.MarshalIndent(config, "", "  ")
	builder.WriteString(string(configBytes))
	builder.WriteString("\n```\n")

	if result.Valid {
		builder.WriteString("\n---\n**Next step:** Remove the `dry_run: true` parameter to actually create this resource.\n")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: builder.String(),
			},
		},
	}
}
