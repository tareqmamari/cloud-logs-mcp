package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResourceValidator(t *testing.T) {
	validator := NewResourceValidator("TestResource", []string{"name", "type"})
	validator.AddFieldValidator("name", FieldValidator{
		Type:      "string",
		MinLength: 1,
		MaxLength: 100,
		Required:  true,
	})
	validator.AddFieldValidator("type", FieldValidator{
		Type:          "string",
		AllowedValues: []string{"a", "b", "c"},
		Required:      true,
	})

	tests := []struct {
		name     string
		config   map[string]interface{}
		wantErr  bool
		errCount int
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"name": "test-resource",
				"type": "a",
			},
			wantErr:  false,
			errCount: 0,
		},
		{
			name: "missing required field",
			config: map[string]interface{}{
				"name": "test-resource",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "invalid enum value",
			config: map[string]interface{}{
				"name": "test-resource",
				"type": "invalid",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name: "name too long",
			config: map[string]interface{}{
				"name": strings.Repeat("a", 101),
				"type": "a",
			},
			wantErr:  true,
			errCount: 1,
		},
		{
			name:     "empty config",
			config:   map[string]interface{}{},
			wantErr:  true,
			errCount: 2, // missing name and type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.config)
			if result.Valid == tt.wantErr {
				t.Errorf("Validate() valid = %v, wantErr %v", result.Valid, tt.wantErr)
			}
			if len(result.Errors) != tt.errCount {
				t.Errorf("Validate() error count = %d, want %d, errors: %v", len(result.Errors), tt.errCount, result.Errors)
			}
		})
	}
}

func TestFieldValidatorTypes(t *testing.T) {
	tests := []struct {
		name      string
		validator FieldValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "valid string",
			validator: FieldValidator{Type: "string", MinLength: 1},
			value:     "hello",
			wantErr:   false,
		},
		{
			name:      "invalid string type",
			validator: FieldValidator{Type: "string"},
			value:     123,
			wantErr:   true,
		},
		{
			name:      "valid int",
			validator: FieldValidator{Type: "int", MinValue: 1, MaxValue: 100},
			value:     float64(50),
			wantErr:   false,
		},
		{
			name:      "int too small",
			validator: FieldValidator{Type: "int", MinValue: 10},
			value:     float64(5),
			wantErr:   true,
		},
		{
			name:      "int too large",
			validator: FieldValidator{Type: "int", MaxValue: 100},
			value:     float64(150),
			wantErr:   true,
		},
		{
			name:      "valid bool",
			validator: FieldValidator{Type: "bool"},
			value:     true,
			wantErr:   false,
		},
		{
			name:      "invalid bool type",
			validator: FieldValidator{Type: "bool"},
			value:     "true",
			wantErr:   true,
		},
		{
			name:      "valid object",
			validator: FieldValidator{Type: "object"},
			value:     map[string]interface{}{"key": "value"},
			wantErr:   false,
		},
		{
			name:      "invalid object type",
			validator: FieldValidator{Type: "object"},
			value:     "not an object",
			wantErr:   true,
		},
		{
			name:      "valid array",
			validator: FieldValidator{Type: "array"},
			value:     []interface{}{"a", "b"},
			wantErr:   false,
		},
		{
			name:      "invalid array type",
			validator: FieldValidator{Type: "array"},
			value:     "not an array",
			wantErr:   true,
		},
		{
			name:      "valid enum",
			validator: FieldValidator{Type: "string", AllowedValues: []string{"a", "b", "c"}},
			value:     "b",
			wantErr:   false,
		},
		{
			name:      "invalid enum",
			validator: FieldValidator{Type: "string", AllowedValues: []string{"a", "b", "c"}},
			value:     "d",
			wantErr:   true,
		},
		{
			name:      "string with pattern match",
			validator: FieldValidator{Type: "string", Pattern: "^[a-z]+$"},
			value:     "hello",
			wantErr:   false,
		},
		{
			name:      "string pattern mismatch",
			validator: FieldValidator{Type: "string", Pattern: "^[a-z]+$"},
			value:     "Hello123",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &ResourceValidator{fieldValidators: make(map[string]FieldValidator)}
			err := rv.validateField("test", tt.value, tt.validator)
			hasErr := err != ""
			if hasErr != tt.wantErr {
				t.Errorf("validateField() error = %q, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid:   true,
		Summary: map[string]interface{}{"name": "test"},
	}

	if !result.Valid {
		t.Error("Expected result to be valid")
	}

	result.Errors = append(result.Errors, "test error")
	result.Valid = false

	if result.Valid {
		t.Error("Expected result to be invalid after adding error")
	}
}

func TestImpactEstimate(t *testing.T) {
	result := &ValidationResult{
		Valid: true,
		EstimatedImpact: &ImpactEstimate{
			AffectedResources: 10,
			EstimatedCost:     "$5/month",
			EstimatedLatency:  "100ms",
			RiskLevel:         "low",
		},
	}

	if result.EstimatedImpact.AffectedResources != 10 {
		t.Errorf("AffectedResources = %d, want 10", result.EstimatedImpact.AffectedResources)
	}
	if result.EstimatedImpact.RiskLevel != "low" {
		t.Errorf("RiskLevel = %s, want low", result.EstimatedImpact.RiskLevel)
	}
}

func TestFormatDryRunResult(t *testing.T) {
	result := &ValidationResult{
		Valid:       true,
		Summary:     map[string]interface{}{"name": "test-resource"},
		Warnings:    []string{"This is a warning"},
		Suggestions: []string{"Consider doing X"},
		EstimatedImpact: &ImpactEstimate{
			RiskLevel: "low",
		},
	}

	config := map[string]interface{}{
		"name": "test-resource",
	}

	mcpResult := FormatDryRunResult(result, "TestResource", config)
	if mcpResult == nil {
		t.Fatal("FormatDryRunResult returned nil")
	}

	if len(mcpResult.Content) == 0 {
		t.Fatal("FormatDryRunResult returned empty content")
	}

	textContent, ok := mcpResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	// Check for expected content
	if !strings.Contains(textContent.Text, "TestResource") {
		t.Error("Expected resource type in output")
	}
	if !strings.Contains(textContent.Text, "Valid") {
		t.Error("Expected status in output")
	}
	if !strings.Contains(textContent.Text, "warning") {
		t.Error("Expected warning in output")
	}
	if !strings.Contains(textContent.Text, "Risk Level") {
		t.Error("Expected risk level in output")
	}
}

func TestFormatDryRunResultInvalid(t *testing.T) {
	result := &ValidationResult{
		Valid:  false,
		Errors: []string{"Missing required field: name"},
	}

	config := map[string]interface{}{}

	mcpResult := FormatDryRunResult(result, "TestResource", config)
	textContent := mcpResult.Content[0].(*mcp.TextContent)

	if !strings.Contains(textContent.Text, "Invalid") {
		t.Error("Expected Invalid status in output")
	}
	if !strings.Contains(textContent.Text, "Missing required field") {
		t.Error("Expected error message in output")
	}
}

func TestValidateRequiredFields(t *testing.T) {
	config := map[string]interface{}{
		"name": "test",
	}

	errors := ValidateRequiredFields(config, []string{"name", "type"})
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	errors = ValidateRequiredFields(config, []string{"name"})
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestValidateEnumField(t *testing.T) {
	config := map[string]interface{}{
		"type": "valid",
	}

	err := ValidateEnumField(config, "type", []string{"valid", "also_valid"})
	if err != "" {
		t.Errorf("Expected no error, got: %s", err)
	}

	err = ValidateEnumField(config, "type", []string{"not_matching"})
	if err == "" {
		t.Error("Expected error for invalid enum value")
	}
}

func TestValidateStringLength(t *testing.T) {
	config := map[string]interface{}{
		"name": "test",
	}

	err := ValidateStringLength(config, "name", 1, 10)
	if err != "" {
		t.Errorf("Expected no error, got: %s", err)
	}

	err = ValidateStringLength(config, "name", 10, 20)
	if err == "" {
		t.Error("Expected error for string too short")
	}

	config["name"] = "this is a very long name"
	err = ValidateStringLength(config, "name", 1, 10)
	if err == "" {
		t.Error("Expected error for string too long")
	}
}

func TestValidateIntRange(t *testing.T) {
	config := map[string]interface{}{
		"count": float64(50),
	}

	err := ValidateIntRange(config, "count", 1, 100)
	if err != "" {
		t.Errorf("Expected no error, got: %s", err)
	}

	err = ValidateIntRange(config, "count", 60, 100)
	if err == "" {
		t.Error("Expected error for int too small")
	}

	err = ValidateIntRange(config, "count", 1, 40)
	if err == "" {
		t.Error("Expected error for int too large")
	}
}
