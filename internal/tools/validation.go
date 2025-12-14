package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ValidationResult represents the result of a dry-run validation
type ValidationResult struct {
	Valid           bool                   `json:"valid"`
	Errors          []string               `json:"errors,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	Summary         map[string]interface{} `json:"summary,omitempty"`
	Suggestions     []string               `json:"suggestions,omitempty"`
	EstimatedImpact *ImpactEstimate        `json:"estimated_impact,omitempty"`
}

// ImpactEstimate provides estimates for the operation's impact
type ImpactEstimate struct {
	AffectedResources int    `json:"affected_resources,omitempty"`
	EstimatedCost     string `json:"estimated_cost,omitempty"`
	EstimatedLatency  string `json:"estimated_latency,omitempty"`
	RiskLevel         string `json:"risk_level,omitempty"` // low, medium, high
}

// DryRunValidator interface for tools that support dry-run validation
type DryRunValidator interface {
	// ValidateConfig validates the configuration without executing
	ValidateConfig(config map[string]interface{}) *ValidationResult
	// GetRequiredFields returns the list of required fields
	GetRequiredFields() []string
	// GetResourceType returns the type of resource being validated
	GetResourceType() string
}

// ResourceValidator provides common validation logic for resources
type ResourceValidator struct {
	resourceType    string
	requiredFields  []string
	fieldValidators map[string]FieldValidator
}

// FieldValidator defines validation rules for a single field
type FieldValidator struct {
	Type          string   // string, int, bool, object, array
	MinLength     int      // for strings
	MaxLength     int      // for strings
	MinValue      int      // for ints
	MaxValue      int      // for ints
	AllowedValues []string // enum values
	Pattern       string   // regex pattern for strings
	Required      bool
}

// NewResourceValidator creates a new resource validator
func NewResourceValidator(resourceType string, requiredFields []string) *ResourceValidator {
	return &ResourceValidator{
		resourceType:    resourceType,
		requiredFields:  requiredFields,
		fieldValidators: make(map[string]FieldValidator),
	}
}

// AddFieldValidator adds a validator for a specific field
func (rv *ResourceValidator) AddFieldValidator(field string, validator FieldValidator) {
	rv.fieldValidators[field] = validator
}

// Validate performs validation on the given config
func (rv *ResourceValidator) Validate(config map[string]interface{}) *ValidationResult {
	result := &ValidationResult{
		Valid:   true,
		Summary: make(map[string]interface{}),
	}

	// Track which fields have been checked as required
	checkedRequired := make(map[string]bool)

	// Check required fields from the requiredFields list
	for _, field := range rv.requiredFields {
		if _, ok := config[field]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
			result.Valid = false
		}
		checkedRequired[field] = true
	}

	// Apply field validators
	for field, validator := range rv.fieldValidators {
		val, exists := config[field]
		if !exists {
			// Only add error if field is required AND not already checked above
			if validator.Required && !checkedRequired[field] {
				result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
				result.Valid = false
			}
			continue
		}

		if err := rv.validateField(field, val, validator); err != "" {
			result.Errors = append(result.Errors, err)
			result.Valid = false
		}
	}

	return result
}

// validateField validates a single field value against its validator
func (rv *ResourceValidator) validateField(field string, val interface{}, v FieldValidator) string {
	switch v.Type {
	case "string":
		return rv.validateStringField(field, val, v)
	case "int":
		return rv.validateIntField(field, val, v)
	case "bool":
		return rv.validateBoolField(field, val)
	case "object":
		return rv.validateObjectField(field, val)
	case "array":
		return rv.validateArrayField(field, val)
	}
	return ""
}

func (rv *ResourceValidator) validateStringField(field string, val interface{}, v FieldValidator) string {
	strVal, ok := val.(string)
	if !ok {
		return fmt.Sprintf("Field '%s' must be a string", field)
	}
	if v.MinLength > 0 && len(strVal) < v.MinLength {
		return fmt.Sprintf("Field '%s' must be at least %d characters", field, v.MinLength)
	}
	if v.MaxLength > 0 && len(strVal) > v.MaxLength {
		return fmt.Sprintf("Field '%s' must be at most %d characters", field, v.MaxLength)
	}
	if len(v.AllowedValues) > 0 && !isAllowedValue(strVal, v.AllowedValues) {
		return fmt.Sprintf("Invalid value for '%s': got '%s', must be one of: %v", field, strVal, v.AllowedValues)
	}
	if v.Pattern != "" {
		matched, err := regexp.MatchString(v.Pattern, strVal)
		if err != nil || !matched {
			return fmt.Sprintf("Field '%s' does not match required pattern", field)
		}
	}
	return ""
}

func isAllowedValue(val string, allowed []string) bool {
	for _, a := range allowed {
		if val == a {
			return true
		}
	}
	return false
}

func (rv *ResourceValidator) validateIntField(field string, val interface{}, v FieldValidator) string {
	intVal, ok := toInt(val)
	if !ok {
		return fmt.Sprintf("Field '%s' must be a number", field)
	}
	if v.MinValue > 0 && intVal < v.MinValue {
		return fmt.Sprintf("Field '%s' must be at least %d", field, v.MinValue)
	}
	if v.MaxValue > 0 && intVal > v.MaxValue {
		return fmt.Sprintf("Field '%s' must be at most %d", field, v.MaxValue)
	}
	return ""
}

func toInt(val interface{}) (int, bool) {
	switch num := val.(type) {
	case float64:
		return int(num), true
	case int:
		return num, true
	case int64:
		return int(num), true
	default:
		return 0, false
	}
}

func (rv *ResourceValidator) validateBoolField(field string, val interface{}) string {
	if _, ok := val.(bool); !ok {
		return fmt.Sprintf("Field '%s' must be a boolean", field)
	}
	return ""
}

func (rv *ResourceValidator) validateObjectField(field string, val interface{}) string {
	if _, ok := val.(map[string]interface{}); !ok {
		return fmt.Sprintf("Field '%s' must be an object", field)
	}
	return ""
}

func (rv *ResourceValidator) validateArrayField(field string, val interface{}) string {
	if _, ok := val.([]interface{}); !ok {
		return fmt.Sprintf("Field '%s' must be an array", field)
	}
	return ""
}

// GetResourceType returns the resource type
func (rv *ResourceValidator) GetResourceType() string {
	return rv.resourceType
}

// GetRequiredFields returns the required fields
func (rv *ResourceValidator) GetRequiredFields() []string {
	return rv.requiredFields
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
func ValidateIntRange(config map[string]interface{}, field string, minVal, maxVal int) string {
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
	if intVal < minVal {
		return fmt.Sprintf("Field '%s' must be at least %d", field, minVal)
	}
	if maxVal > 0 && intVal > maxVal {
		return fmt.Sprintf("Field '%s' must be at most %d", field, maxVal)
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

	// Add impact estimate if available
	if result.EstimatedImpact != nil {
		builder.WriteString("### Estimated Impact\n\n")
		if result.EstimatedImpact.AffectedResources > 0 {
			builder.WriteString(fmt.Sprintf("- **Affected Resources:** %d\n", result.EstimatedImpact.AffectedResources))
		}
		if result.EstimatedImpact.EstimatedCost != "" {
			builder.WriteString(fmt.Sprintf("- **Estimated Cost:** %s\n", result.EstimatedImpact.EstimatedCost))
		}
		if result.EstimatedImpact.EstimatedLatency != "" {
			builder.WriteString(fmt.Sprintf("- **Estimated Latency:** %s\n", result.EstimatedImpact.EstimatedLatency))
		}
		if result.EstimatedImpact.RiskLevel != "" {
			riskEmoji := "🟢"
			switch result.EstimatedImpact.RiskLevel {
			case "medium":
				riskEmoji = "🟡"
			case "high":
				riskEmoji = "🔴"
			}
			builder.WriteString(fmt.Sprintf("- **Risk Level:** %s %s\n", riskEmoji, result.EstimatedImpact.RiskLevel))
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
