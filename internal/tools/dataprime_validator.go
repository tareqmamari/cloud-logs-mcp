package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// DataPrimeValidationError represents a validation error with helpful guidance
type DataPrimeValidationError struct {
	Message    string
	Suggestion string
	Field      string // The problematic field if applicable
}

func (e *DataPrimeValidationError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s\n\n💡 **Suggestion:** %s", e.Message, e.Suggestion)
	}
	return e.Message
}

// ValidateDataPrimeQuery validates a DataPrime query and returns helpful errors
func ValidateDataPrimeQuery(query string) *DataPrimeValidationError {
	// Check for common syntax errors in order of likelihood

	// 1. Check for ~~ operator which is NOT valid DataPrime syntax
	if err := validateNoTildeOperator(query); err != nil {
		return err
	}

	// 2. Check for invalid field references
	if err := validateFieldReferences(query); err != nil {
		return err
	}

	// 3. Check for common typos and mistakes
	if err := validateCommonMistakes(query); err != nil {
		return err
	}

	// 4. Validate severity values if used (but numeric values are now auto-corrected)
	if err := validateSeverityUsage(query); err != nil {
		return err
	}

	return nil
}

// AutoCorrectDataPrimeQuery fixes common issues in DataPrime queries automatically.
// This improves UX by fixing known issues instead of rejecting queries.
// Returns the corrected query and a list of corrections made.
func AutoCorrectDataPrimeQuery(query string) (string, []string) {
	var corrections []string
	corrected := query

	// Auto-correct numeric severity to named severity
	// Maps: 0→VERBOSE, 1→DEBUG, 2→INFO, 3→WARNING, 4→ERROR, 5→CRITICAL
	severityNumToName := map[string]string{
		"0": "VERBOSE",
		"1": "DEBUG",
		"2": "INFO",
		"3": "WARNING",
		"4": "ERROR",
		"5": "CRITICAL",
		"6": "CRITICAL", // Handle out-of-range as CRITICAL
	}

	numericSeverityPattern := regexp.MustCompile(`(\$m\.severity\s*(?:==|!=|>=|<=|>|<)\s*)(\d+)`)
	if matches := numericSeverityPattern.FindAllStringSubmatch(corrected, -1); len(matches) > 0 {
		for _, match := range matches {
			numValue := match[2]
			namedValue := severityNumToName[numValue]
			if namedValue == "" {
				namedValue = "ERROR" // Default for unknown values
			}
			oldVal := match[0]
			newVal := match[1] + namedValue
			corrected = strings.Replace(corrected, oldVal, newVal, 1)
			corrections = append(corrections, fmt.Sprintf("severity %s → %s", numValue, namedValue))
		}
	}

	// Auto-correct common severity typos
	severityTypos := map[string]string{
		"ERR":      "ERROR",
		"WARN":     "WARNING",
		"CRIT":     "CRITICAL",
		"DBG":      "DEBUG",
		"VERB":     "VERBOSE",
		"error":    "ERROR",
		"warning":  "WARNING",
		"critical": "CRITICAL",
		"info":     "INFO",
		"debug":    "DEBUG",
	}

	for typo, correct := range severityTypos {
		typoPattern := regexp.MustCompile(`(\$m\.severity\s*(?:==|!=|>=|<=|>|<)\s*)` + regexp.QuoteMeta(typo) + `\b`)
		if typoPattern.MatchString(corrected) {
			corrected = typoPattern.ReplaceAllString(corrected, "${1}"+correct)
			corrections = append(corrections, fmt.Sprintf("severity %s → %s", typo, correct))
		}
	}

	return corrected, corrections
}

// validateNoTildeOperator checks that ~~ is NOT used (it's not valid DataPrime)
// DataPrime uses matches() for regex and contains() for substring matching
func validateNoTildeOperator(query string) *DataPrimeValidationError {
	// The ~~ operator is NOT valid DataPrime syntax!
	// Users should use matches() for regex or contains() for substring matching
	tildePattern := regexp.MustCompile(`\s*!?~~\s*`)

	if tildePattern.MatchString(query) {
		return &DataPrimeValidationError{
			Message: "The ~~ operator is not valid DataPrime syntax",
			Suggestion: `DataPrime uses functions for pattern matching:

**For substring matching:**
- field.contains('substring')
- contains(field, 'substring')

**For regex matching:**
- field.matches(/pattern/)
- matches(field, /pattern/)

**For free-text search across all fields:**
- Use Lucene syntax: lucene 'error AND timeout'
- Or: find 'error' in message

**Examples:**
- filter $d.message.contains('error')
- filter $l.applicationname.startsWith('prod-')
- filter message.matches(/error.*timeout/)`,
		}
	}

	return nil
}

// validateFieldReferences checks for invalid or non-existent field references
func validateFieldReferences(query string) *DataPrimeValidationError {
	// Known valid label fields in IBM Cloud Logs / Coralogix
	validLabelFields := map[string]bool{
		"applicationname": true,
		"subsystemname":   true,
		"computername":    true,
		"ipaddress":       true,
		"threadid":        true,
		"processid":       true,
		"classname":       true,
		"methodname":      true,
		"category":        true,
	}

	// Known valid metadata fields
	validMetadataFields := map[string]bool{
		"severity":  true,
		"timestamp": true,
		"priority":  true,
	}

	// Common invalid field names that users might try with helpful suggestions
	invalidFieldSuggestions := map[string]string{
		"namespace":   "Use $l.applicationname instead - Kubernetes namespace is typically stored in applicationname",
		"pod":         "Pod name is usually in $d. (data fields) or part of applicationname/subsystemname. Check your log structure.",
		"container":   "Container name is usually in $d. (data fields) or part of subsystemname. Check your log structure.",
		"node":        "Node name is usually in $d. (data fields). Check your log structure.",
		"cluster":     "Cluster name is usually in $d. (data fields). Check your log structure.",
		"environment": "Environment is usually in $d. (data fields) or part of applicationname. Check your log structure.",
		"service":     "Use $l.applicationname or $l.subsystemname - service maps to these fields",
		"level":       "Use $m.severity for log level. Valid values: DEBUG, VERBOSE, INFO, WARNING, ERROR, CRITICAL",
		"loglevel":    "Use $m.severity for log level. Valid values: DEBUG, VERBOSE, INFO, WARNING, ERROR, CRITICAL",
		"log_level":   "Use $m.severity for log level. Valid values: DEBUG, VERBOSE, INFO, WARNING, ERROR, CRITICAL",
		"message":     "Message content is in $d. (data fields). Use $d.message, $d.msg, $d.log, or check your log structure.",
		"msg":         "Message content is in $d. (data fields). Use $d.message, $d.msg, $d.log, or check your log structure.",
		"text":        "Use $d. fields for log content (e.g., $d.message). For free-text search, use: lucene 'your search term'",
	}

	// Find all $l. field references
	labelFieldPattern := regexp.MustCompile(`\$l\.(\w+)`)
	labelMatches := labelFieldPattern.FindAllStringSubmatch(query, -1)
	for _, match := range labelMatches {
		field := strings.ToLower(match[1])
		if !validLabelFields[field] {
			suggestion := invalidFieldSuggestions[field]
			if suggestion == "" {
				suggestion = fmt.Sprintf("Valid label fields are: applicationname, subsystemname, computername, ipaddress, threadid, processid, classname, methodname, category.\n\nIf '%s' is a custom field, it's likely in $d. (data) instead.", field)
			}
			return &DataPrimeValidationError{
				Message:    fmt.Sprintf("Unknown label field: $l.%s", field),
				Suggestion: suggestion,
				Field:      "$l." + field,
			}
		}
	}

	// Find all $m. field references
	metadataFieldPattern := regexp.MustCompile(`\$m\.(\w+)`)
	metadataMatches := metadataFieldPattern.FindAllStringSubmatch(query, -1)
	for _, match := range metadataMatches {
		field := strings.ToLower(match[1])
		if !validMetadataFields[field] {
			suggestion := invalidFieldSuggestions[field]
			if suggestion == "" {
				suggestion = fmt.Sprintf("Valid metadata fields are: severity, timestamp, priority.\n\nIf '%s' is a custom field, it's likely in $d. (data) or $l. (labels) instead.", field)
			}
			return &DataPrimeValidationError{
				Message:    fmt.Sprintf("Unknown metadata field: $m.%s", field),
				Suggestion: suggestion,
				Field:      "$m." + field,
			}
		}
	}

	return nil
}

// validateSeverityUsage checks that severity values are valid
func validateSeverityUsage(query string) *DataPrimeValidationError {
	// Valid severity levels in DataPrime
	validSeverities := map[string]bool{
		"VERBOSE":  true,
		"DEBUG":    true,
		"INFO":     true,
		"WARNING":  true,
		"ERROR":    true,
		"CRITICAL": true,
	}

	// Note: Numeric severity values are now auto-corrected by AutoCorrectDataPrimeQuery()
	// so we don't reject them here. This validation now only catches truly invalid names.

	// Check for $m.severity with invalid named value
	severityPattern := regexp.MustCompile(`\$m\.severity\s*(==|!=|>=|<=|>|<)\s*['"]?([a-zA-Z]+)['"]?`)
	matches := severityPattern.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		value := strings.ToUpper(match[2])
		if !validSeverities[value] {
			return &DataPrimeValidationError{
				Message: fmt.Sprintf("Invalid severity value: %s", match[2]),
				Suggestion: `Valid severity levels are: VERBOSE, DEBUG, INFO, WARNING, ERROR, CRITICAL

**Severity order (low to high):**
VERBOSE < DEBUG < INFO < WARNING < ERROR < CRITICAL

**Examples:**
- filter $m.severity == ERROR
- filter $m.severity >= WARNING
- filter $m.severity != DEBUG`,
				Field: "$m.severity",
			}
		}
	}

	return nil
}

// validateCommonMistakes checks for common syntax mistakes
func validateCommonMistakes(query string) *DataPrimeValidationError {
	// Check for = instead of == for comparison
	// Pattern: $x.field = 'value' (but not $x.field == 'value')
	singleEqualsPattern := regexp.MustCompile(`\$[dlmp]\.\w+\s*=\s*[^=]`)
	if singleEqualsPattern.MatchString(query) {
		return &DataPrimeValidationError{
			Message:    "Single = is not valid for comparison in DataPrime",
			Suggestion: "Use == for equality comparison.\n\nExample: `filter $l.applicationname == 'myapp'`",
		}
	}

	// Check for AND/OR instead of &&/||
	if regexp.MustCompile(`\s+AND\s+`).MatchString(strings.ToUpper(query)) {
		return &DataPrimeValidationError{
			Message:    "Use && instead of AND for logical AND in DataPrime",
			Suggestion: "Example: `filter $l.applicationname == 'app1' && $m.severity >= WARNING`",
		}
	}
	if regexp.MustCompile(`\s+OR\s+`).MatchString(strings.ToUpper(query)) {
		return &DataPrimeValidationError{
			Message:    "Use || instead of OR for logical OR in DataPrime",
			Suggestion: "Example: `filter $l.applicationname == 'app1' || $l.applicationname == 'app2'`",
		}
	}

	// Check for common field name typos
	fieldPattern := regexp.MustCompile(`\$([lm])\.(\w+)`)
	matches := fieldPattern.FindAllStringSubmatch(query, -1)

	typos := map[string]string{
		"application":    "applicationname",
		"app":            "applicationname",
		"subsystem":      "subsystemname",
		"appname":        "applicationname",
		"app_name":       "applicationname",
		"subsystem_name": "subsystemname",
		"level":          "severity (in $m.)",
		"log_level":      "severity (in $m.)",
		"loglevel":       "severity (in $m.)",
	}

	for _, match := range matches {
		prefix := match[1]
		field := strings.ToLower(match[2])
		if correct, isTypo := typos[field]; isTypo {
			fullField := fmt.Sprintf("$%s.%s", prefix, field)
			if strings.Contains(correct, "$m.") {
				return &DataPrimeValidationError{
					Message:    fmt.Sprintf("Invalid field reference: %s", fullField),
					Suggestion: "Use $m.severity for log level instead",
					Field:      fullField,
				}
			}
			return &DataPrimeValidationError{
				Message:    fmt.Sprintf("Invalid field reference: %s", fullField),
				Suggestion: fmt.Sprintf("Use $%s.%s instead", prefix, correct),
				Field:      fullField,
			}
		}
	}

	// Check for quotes issues - using double quotes instead of single
	// DataPrime uses single quotes for strings
	doubleQuotesPattern := regexp.MustCompile(`\$[dlmp]\.\w+\s*==\s*"[^"]*"`)
	if doubleQuotesPattern.MatchString(query) {
		return &DataPrimeValidationError{
			Message:    "Use single quotes for string values in DataPrime, not double quotes",
			Suggestion: "Example: `filter $l.applicationname == 'myapp'` (single quotes)",
		}
	}

	// Check for LIKE keyword (SQL syntax, not DataPrime)
	if regexp.MustCompile(`\s+LIKE\s+`).MatchString(strings.ToUpper(query)) {
		return &DataPrimeValidationError{
			Message: "LIKE is not valid DataPrime syntax",
			Suggestion: `Use DataPrime string functions instead:
- contains(): field.contains('substring')
- startsWith(): field.startsWith('prefix')
- endsWith(): field.endsWith('suffix')
- matches(): field.matches(/regex/)`,
		}
	}

	// Check for IN keyword (SQL syntax, not DataPrime)
	if regexp.MustCompile(`\s+IN\s*\(`).MatchString(strings.ToUpper(query)) {
		return &DataPrimeValidationError{
			Message: "IN is not valid DataPrime syntax",
			Suggestion: `Use multiple OR conditions with || instead:
filter $l.applicationname == 'app1' || $l.applicationname == 'app2' || $l.applicationname == 'app3'`,
		}
	}

	return nil
}

// SuggestQueryFix attempts to suggest a fix for a query that failed
func SuggestQueryFix(_ string, errorMessage string) string {
	lowerError := strings.ToLower(errorMessage)

	// Handle "~~ only works on $d" error (even though we validate against this)
	if strings.Contains(lowerError, "~~") {
		return `The ~~ operator is not standard DataPrime syntax. Use functions instead:

**For substring matching:**
- field.contains('substring')
- filter $d.message.contains('error')

**For regex matching:**
- field.matches(/pattern/)
- filter $d.message.matches(/error.*timeout/)

**For free-text search:**
- Use Lucene: lucene 'error AND timeout'
- Or: find 'error' in message`
	}

	// Handle "keypath does not exist" error
	if strings.Contains(lowerError, "keypath does not exist") {
		return `This field doesn't exist in your logs. Common causes:

1. **Wrong prefix**: Check if the field should be:
   - $l. (labels): applicationname, subsystemname
   - $m. (metadata): severity, timestamp, priority
   - $d. (data): your log payload fields

2. **Field doesn't exist**: The field may not be present in logs for this time range

3. **Case sensitivity**: Field names are case-sensitive

**Tip**: Run 'source logs | limit 10' to see available fields in your logs.`
	}

	// Handle compilation errors
	if strings.Contains(lowerError, "compilation error") {
		return `Query syntax error. Common fixes:

1. Use == for comparison (not = or LIKE)
2. Use && for AND, || for OR (not AND/OR keywords)
3. Use single quotes for strings: 'value' (not "value")
4. Include 'source logs' at the beginning
5. Use proper field prefixes: $l., $m., $d.
6. Use contains() or matches() for pattern matching (not ~~)`
	}

	// Handle unknown function errors
	if strings.Contains(lowerError, "unknown function") {
		return `Unknown function. Common DataPrime functions:

**Aggregation:** count(), sum(), avg(), min(), max(), distinct_count()
**String:** contains(), startsWith(), endsWith(), matches(), toLowerCase(), toUpperCase(), trim(), concat()
**Time:** now(), parseTimestamp(), formatTimestamp(), diffTime()
**Conditional:** if(), case {}, coalesce()`
	}

	return ""
}

// FormatQueryError creates a user-friendly error message for query failures
func FormatQueryError(query string, apiError string) string {
	var sb strings.Builder

	sb.WriteString("**Query Error**\n\n")
	sb.WriteString(fmt.Sprintf("Query: `%s`\n\n", truncateQueryForDisplay(query, 200)))
	sb.WriteString(fmt.Sprintf("Error: %s\n\n", apiError))

	suggestion := SuggestQueryFix(query, apiError)
	if suggestion != "" {
		sb.WriteString("---\n\n")
		sb.WriteString(suggestion)
	}

	return sb.String()
}

// truncateQueryForDisplay truncates a query for error display
func truncateQueryForDisplay(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen-3] + "..."
}

// GetDataPrimeQuickReference returns a quick reference for DataPrime syntax
func GetDataPrimeQuickReference() string {
	return `# DataPrime Quick Reference

## Field Access
- $d.field or just field - User data (log payload)
- $l.applicationname - Labels
- $m.severity - Metadata

## Operators
- == != > < >= <= - Comparison
- && || ! - Logical (NOT: AND, OR)

## String Functions (NOT ~~)
- field.contains('text')
- field.startsWith('prefix')
- field.endsWith('suffix')
- field.matches(/regex/)
- field.toLowerCase()

## Commands
source logs | filter <cond> | groupby <field> aggregate <func> | limit <n>

## Severity Levels
VERBOSE < DEBUG < INFO < WARNING < ERROR < CRITICAL
`
}
