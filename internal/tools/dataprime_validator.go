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

// injectionPatterns contains regex patterns for detecting potential injection attacks.
// These patterns detect SQL-like injection attempts that could be maliciously crafted.
var injectionPatterns = []*regexp.Regexp{
	// SQL-style command injection (case-insensitive)
	regexp.MustCompile(`(?i);\s*(DROP|DELETE|TRUNCATE|ALTER|CREATE|INSERT|UPDATE|EXEC|EXECUTE)\s+`),
	// UNION-based injection
	regexp.MustCompile(`(?i)\bUNION\s+(ALL\s+)?SELECT\b`),
	// Comment-based injection (SQL comments that might bypass filters)
	regexp.MustCompile(`--\s*$`),                    // Trailing SQL comment
	regexp.MustCompile(`/\*[\s\S]*?\*/`),            // Block comments used for obfuscation
	regexp.MustCompile(`(?i);\s*--`),                // Statement termination with comment
	regexp.MustCompile(`(?i)'\s*;\s*--`),            // String escape with statement termination
	regexp.MustCompile(`(?i)'\s*OR\s+'1'\s*=\s*'1`), // Classic OR injection
	regexp.MustCompile(`(?i)'\s*OR\s+1\s*=\s*1`),    // Numeric OR injection
	// Command execution attempts
	regexp.MustCompile(`(?i)\bEXEC\s*\(`),          // EXEC() calls
	regexp.MustCompile(`(?i)\bxp_cmdshell\b`),      // SQL Server command execution
	regexp.MustCompile(`(?i)\bsys\.(dm_|fn_|sp_)`), // System object access
	// Stacked queries / multiple statements
	regexp.MustCompile(`;\s*SELECT\s+`), // Stacked SELECT
	regexp.MustCompile(`;\s*INSERT\s+`), // Stacked INSERT
	regexp.MustCompile(`;\s*UPDATE\s+`), // Stacked UPDATE
	regexp.MustCompile(`;\s*DELETE\s+`), // Stacked DELETE
}

// validateNoInjectionPatterns checks for potential injection attack patterns.
// This provides defense-in-depth against maliciously crafted queries.
func validateNoInjectionPatterns(query string) *DataPrimeValidationError {
	for _, pattern := range injectionPatterns {
		if pattern.MatchString(query) {
			return &DataPrimeValidationError{
				Message: "Query contains potentially unsafe patterns",
				Suggestion: `The query contains patterns that resemble injection attacks and has been rejected for security reasons.

If this is a legitimate query, please:
1. Remove any SQL-style comments (-- or /* */)
2. Avoid semicolons followed by SQL keywords
3. Use DataPrime syntax instead of SQL syntax

**DataPrime uses different syntax:**
- No semicolons between statements (use | pipe instead)
- No SQL keywords like UNION, DROP, DELETE, etc.
- Comments are not supported in queries

If you believe this is a false positive, please contact support.`,
			}
		}
	}

	return nil
}

// ValidateDataPrimeQuery validates a DataPrime query and returns helpful errors
func ValidateDataPrimeQuery(query string) *DataPrimeValidationError {
	// Check for common syntax errors in order of likelihood

	// 0. Check for injection patterns (security first)
	if err := validateNoInjectionPatterns(query); err != nil {
		return err
	}

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

// mixedTypeFields are $d fields that commonly have object|string type and need casting
// when using string methods like .contains(), .startsWith(), .endsWith(), .matches()
var mixedTypeFields = map[string]bool{
	"message": true,
	"msg":     true,
	"log":     true,
	"text":    true,
	"body":    true,
	"content": true,
	"payload": true,
	"data":    true,
	"error":   true,
	"err":     true,
	"reason":  true,
	"details": true,
}

// stringMethods are DataPrime string methods that require string type input
var stringMethods = []string{"contains", "startsWith", "endsWith", "matches", "toLowerCase", "toUpperCase", "trim"}

// autoCorrectMixedTypeStringMethods adds :string type cast to $d fields that commonly have
// mixed object|string type when string methods are called on them.
// Example: $d.message.contains('error') -> $d.message:string.contains('error')
func autoCorrectMixedTypeStringMethods(query string, corrections []string) (string, []string) {
	corrected := query

	// Build pattern to match $d.field.stringMethod where field is a known mixed-type field
	// Pattern: $d.field.method( where field is in mixedTypeFields and method is a string method
	for field := range mixedTypeFields {
		for _, method := range stringMethods {
			// Match $d.field.method( but not $d.field:string.method(
			pattern := regexp.MustCompile(`(\$d\.` + regexp.QuoteMeta(field) + `)\.` + regexp.QuoteMeta(method) + `\(`)
			if pattern.MatchString(corrected) {
				// Replace with $d.field:string.method(
				replacement := "${1}:string." + method + "("
				corrected = pattern.ReplaceAllString(corrected, replacement)
				corrections = append(corrections, fmt.Sprintf("$d.%s.%s() → $d.%s:string.%s() (field may have mixed object|string type)", field, method, field, method))
			}
		}
	}

	return corrected, corrections
}

// AutoCorrectDataPrimeQuery fixes common issues in DataPrime queries automatically.
// This improves UX by fixing known issues instead of rejecting queries.
// Returns the corrected query and a list of corrections made.
func AutoCorrectDataPrimeQuery(query string) (string, []string) {
	var corrections []string
	corrected := query

	// Apply corrections in order - each returns the corrected query and appends to corrections
	corrected, corrections = autoCorrectMixedTypeStringMethods(corrected, corrections)
	corrected, corrections = autoCorrectLevelToSeverity(corrected, corrections)
	corrected, corrections = autoCorrectStatusCodes(corrected, corrections)
	corrected, corrections = autoCorrectNumericSeverity(corrected, corrections)
	corrected, corrections = autoCorrectSeverityTypos(corrected, corrections)
	corrected, corrections = autoCorrectSortToOrderby(corrected, corrections)
	corrected, corrections = autoCorrectAggregateSyntax(corrected, corrections)
	corrected, corrections = autoCorrectDistinctCount(corrected, corrections)
	corrected, corrections = autoCorrectTimeBucketing(corrected, corrections)

	return corrected, corrections
}

// autoCorrectLevelToSeverity corrects $d.level to $m.severity
func autoCorrectLevelToSeverity(query string, corrections []string) (string, []string) {
	levelToSeverity := map[string]string{
		"'INFO'": "INFO", "'info'": "INFO", "'DEBUG'": "DEBUG", "'debug'": "DEBUG",
		"'WARNING'": "WARNING", "'warning'": "WARNING", "'WARN'": "WARNING", "'warn'": "WARNING",
		"'ERROR'": "ERROR", "'error'": "ERROR", "'ERR'": "ERROR", "'err'": "ERROR",
		"'CRITICAL'": "CRITICAL", "'critical'": "CRITICAL", "'VERBOSE'": "VERBOSE", "'verbose'": "VERBOSE",
	}

	pattern := regexp.MustCompile(`\$d\.level\s*(==|!=)\s*('(?:INFO|info|DEBUG|debug|WARNING|warning|WARN|warn|ERROR|error|ERR|err|CRITICAL|critical|VERBOSE|verbose)')`)
	for _, match := range pattern.FindAllStringSubmatch(query, -1) {
		if severityName := levelToSeverity[match[2]]; severityName != "" {
			oldExpr, newExpr := match[0], "$m.severity "+match[1]+" "+severityName
			query = strings.Replace(query, oldExpr, newExpr, 1)
			corrections = append(corrections, fmt.Sprintf("$d.level %s %s → $m.severity %s %s (level field has mixed types)", match[1], match[2], match[1], severityName))
		}
	}
	return query, corrections
}

// autoCorrectStatusCodes corrects numeric status codes to strings
func autoCorrectStatusCodes(query string, corrections []string) (string, []string) {
	pattern := regexp.MustCompile(`(\$d\.(?:status_code|statusCode|http_status|httpStatus|response_code|responseCode))\s*(==|!=)\s*(\d{3})`)
	for _, match := range pattern.FindAllStringSubmatch(query, -1) {
		oldExpr := match[0]
		newExpr := match[1] + " " + match[2] + " '" + match[3] + "'"
		query = strings.Replace(query, oldExpr, newExpr, 1)
		corrections = append(corrections, fmt.Sprintf("%s %s %s → %s %s '%s' (status codes are often strings)", match[1], match[2], match[3], match[1], match[2], match[3]))
	}
	return query, corrections
}

// autoCorrectNumericSeverity corrects numeric severity to named severity
func autoCorrectNumericSeverity(query string, corrections []string) (string, []string) {
	severityNumToName := map[string]string{"0": "VERBOSE", "1": "DEBUG", "2": "INFO", "3": "WARNING", "4": "ERROR", "5": "CRITICAL", "6": "CRITICAL"}
	pattern := regexp.MustCompile(`(\$m\.severity\s*(?:==|!=|>=|<=|>|<)\s*)(\d+)`)
	for _, match := range pattern.FindAllStringSubmatch(query, -1) {
		namedValue := severityNumToName[match[2]]
		if namedValue == "" {
			namedValue = "ERROR"
		}
		query = strings.Replace(query, match[0], match[1]+namedValue, 1)
		corrections = append(corrections, fmt.Sprintf("severity %s → %s", match[2], namedValue))
	}
	return query, corrections
}

// autoCorrectSeverityTypos corrects common severity typos
func autoCorrectSeverityTypos(query string, corrections []string) (string, []string) {
	typos := map[string]string{
		"ERR": "ERROR", "WARN": "WARNING", "CRIT": "CRITICAL", "DBG": "DEBUG", "VERB": "VERBOSE",
		"error": "ERROR", "warning": "WARNING", "critical": "CRITICAL", "info": "INFO", "debug": "DEBUG",
	}
	for typo, correct := range typos {
		pattern := regexp.MustCompile(`(\$m\.severity\s*(?:==|!=|>=|<=|>|<)\s*)` + regexp.QuoteMeta(typo) + `\b`)
		if pattern.MatchString(query) {
			query = pattern.ReplaceAllString(query, "${1}"+correct)
			corrections = append(corrections, fmt.Sprintf("severity %s → %s", typo, correct))
		}
	}
	return query, corrections
}

// autoCorrectSortToOrderby corrects 'sort' to 'orderby'
func autoCorrectSortToOrderby(query string, corrections []string) (string, []string) {
	pattern := regexp.MustCompile(`\|\s*sort\s+(-?)([a-zA-Z_$][a-zA-Z0-9_.]*)`)
	for _, match := range pattern.FindAllStringSubmatch(query, -1) {
		descending, field := match[1], match[2]
		newExpr := "| orderby " + field
		if descending == "-" {
			newExpr += " desc"
		}
		query = strings.Replace(query, match[0], newExpr, 1)
		desc := ""
		if descending == "-" {
			desc = " desc"
		}
		corrections = append(corrections, fmt.Sprintf("sort %s%s → orderby %s%s (use 'orderby' in DataPrime)", descending, field, field, desc))
	}
	return query, corrections
}

// autoCorrectAggregateSyntax corrects "aggregate alias = func()" to "aggregate func() as alias"
func autoCorrectAggregateSyntax(query string, corrections []string) (string, []string) {
	pattern := regexp.MustCompile(`aggregate\s+(\w+)\s*=\s*(count|sum|avg|min|max|approx_count_distinct|percentile|stddev|variance|any_value|collect)\s*\(([^)]*)\)`)
	for _, match := range pattern.FindAllStringSubmatch(query, -1) {
		alias, funcName, args := match[1], match[2], match[3]
		newExpr := fmt.Sprintf("aggregate %s(%s) as %s", funcName, args, alias)
		query = strings.Replace(query, match[0], newExpr, 1)
		corrections = append(corrections, fmt.Sprintf("aggregate %s = %s() → aggregate %s() as %s (use 'as' for alias)", alias, funcName, funcName, alias))
	}
	return query, corrections
}

// autoCorrectDistinctCount corrects count_distinct/distinct_count to approx_count_distinct
func autoCorrectDistinctCount(query string, corrections []string) (string, []string) {
	pattern := regexp.MustCompile(`(?:^|[^a-zA-Z_])(count_distinct|distinct_count)\s*\(`)
	if matches := pattern.FindAllStringSubmatch(query, -1); len(matches) > 0 {
		for _, match := range matches {
			query = strings.ReplaceAll(query, match[1]+"(", "approx_count_distinct(")
		}
		corrections = append(corrections, "count_distinct/distinct_count → approx_count_distinct (only approx variant supported)")
	}
	return query, corrections
}

// autoCorrectTimeBucketing corrects various invalid time bucketing syntaxes to roundTime()
func autoCorrectTimeBucketing(query string, corrections []string) (string, []string) {
	// Time bucket patterns to correct to roundTime($m.timestamp, interval)
	patterns := []struct {
		re     *regexp.Regexp
		format string // format string for correction message
	}{
		{regexp.MustCompile(`\$m\.timestamp:(\d+[smhd])`), "$m.timestamp:%s → roundTime($m.timestamp, %s)"},
		{regexp.MustCompile(`bucket\s*\(\s*(\$m\.timestamp)\s*,\s*(\d+[smhd])\s*\)`), "bucket(%s, %s) → roundTime(%s, %s)"},
		{regexp.MustCompile(`bin\s*\(\s*(\$m\.timestamp)\s*,\s*'?(\d+[smhd])'?\s*\)`), "bin(%s, %s) → roundTime(%s, %s)"},
		{regexp.MustCompile(`timestamp_round_down\s*\(\s*(\$m\.timestamp)\s*,\s*'?(\d+[smhd])'?\s*\)`), "timestamp_round_down(%s, %s) → roundTime(%s, %s)"},
	}

	for _, p := range patterns {
		for _, match := range p.re.FindAllStringSubmatch(query, -1) {
			var timestamp, interval string
			if len(match) == 2 { // Colon syntax: $m.timestamp:1h
				timestamp, interval = "$m.timestamp", match[1]
			} else { // Function syntax: func($m.timestamp, 1h)
				timestamp, interval = match[1], match[2]
			}
			newExpr := fmt.Sprintf("roundTime(%s, %s)", timestamp, interval)
			query = strings.Replace(query, match[0], newExpr, 1)
			corrections = append(corrections, fmt.Sprintf(p.format+" (use roundTime for time bucketing)", timestamp, interval, timestamp, interval))
		}
	}

	// time_bucket() has reversed argument order
	timeBucketPattern := regexp.MustCompile(`time_bucket\s*\(\s*'?(\d+[smhd])'?\s*,\s*(\$m\.timestamp)\s*\)`)
	for _, match := range timeBucketPattern.FindAllStringSubmatch(query, -1) {
		interval, timestamp := match[1], match[2]
		newExpr := fmt.Sprintf("roundTime(%s, %s)", timestamp, interval)
		query = strings.Replace(query, match[0], newExpr, 1)
		corrections = append(corrections, fmt.Sprintf("time_bucket('%s', %s) → roundTime(%s, %s) (use roundTime for time bucketing)", interval, timestamp, timestamp, interval))
	}

	// date_trunc() and trunc() use unit names instead of intervals
	intervalMap := map[string]string{"hour": "1h", "day": "1d", "minute": "1m", "second": "1s"}

	dateTruncPattern := regexp.MustCompile(`date_trunc\s*\(\s*'(hour|day|minute|second)'\s*,\s*(\$m\.timestamp)\s*\)`)
	for _, match := range dateTruncPattern.FindAllStringSubmatch(query, -1) {
		unit, timestamp := match[1], match[2]
		interval := intervalMap[unit]
		newExpr := fmt.Sprintf("roundTime(%s, %s)", timestamp, interval)
		query = strings.Replace(query, match[0], newExpr, 1)
		corrections = append(corrections, fmt.Sprintf("date_trunc('%s', %s) → roundTime(%s, %s) (use roundTime for time bucketing)", unit, timestamp, timestamp, interval))
	}

	truncPattern := regexp.MustCompile(`trunc\s*\(\s*(\$m\.timestamp)\s*,\s*'(hour|day|minute|second)'\s*\)`)
	for _, match := range truncPattern.FindAllStringSubmatch(query, -1) {
		timestamp, unit := match[1], match[2]
		interval := intervalMap[unit]
		newExpr := fmt.Sprintf("roundTime(%s, %s)", timestamp, interval)
		query = strings.Replace(query, match[0], newExpr, 1)
		corrections = append(corrections, fmt.Sprintf("trunc(%s, '%s') → roundTime(%s, %s) (use roundTime for time bucketing)", timestamp, unit, timestamp, interval))
	}

	// Remove invalid timestampformat stage
	timestampFormatPattern := regexp.MustCompile(`\|\s*timestampformat\s+\$m\.timestamp\s+to_timestamp\s+'[^']+'\s+as\s+(\w+)\s*`)
	for _, match := range timestampFormatPattern.FindAllStringSubmatch(query, -1) {
		query = strings.Replace(query, match[0], " ", 1)
		corrections = append(corrections, fmt.Sprintf("Removed invalid 'timestampformat' stage. Use 'groupby roundTime($m.timestamp, 1h) as %s' for time bucketing", match[1]))
	}
	// Clean up multiple spaces and normalize pipes
	query = regexp.MustCompile(`\s+\|`).ReplaceAllString(query, " |")
	query = regexp.MustCompile(`\|\s+\|`).ReplaceAllString(query, "|")

	return query, corrections
}

// PrepareQuery validates, corrects, and prepares a DataPrime query for execution.
// This is the central entry point that all query execution paths should use.
// It combines auto-correction with validation to provide a consistent query preparation pipeline.
//
// Parameters:
//   - query: The DataPrime query to prepare
//   - tier: The execution tier ("archive" or "frequent_search") - affects some corrections
//   - syntax: The query syntax ("dataprime", "dataprime_utf8_base64", "lucene", etc.)
//
// Returns:
//   - The corrected query string
//   - A list of corrections that were made
//   - An error if the query is invalid after corrections
func PrepareQuery(query string, tier string, syntax string) (string, []string, error) {
	// Skip validation for non-DataPrime queries
	if syntax != "dataprime" && syntax != "dataprime_utf8_base64" && syntax != "" {
		return query, nil, nil
	}

	// 0. Sanitize query - remove/replace problematic characters that can break JSON serialization
	corrected := sanitizeQuery(query)

	// 1. Auto-correct common issues
	corrected, corrections := AutoCorrectDataPrimeQuery(corrected)

	// 2. Apply tier-specific corrections
	corrected, tierCorrections := applyTierCorrections(corrected, tier)
	corrections = append(corrections, tierCorrections...)

	// 3. Validate the final query
	if err := ValidateDataPrimeQuery(corrected); err != nil {
		return "", nil, err
	}

	return corrected, corrections, nil
}

// sanitizeQuery removes or replaces problematic characters that can cause JSON serialization issues
func sanitizeQuery(query string) string {
	// Normalize whitespace - replace tabs and multiple spaces with single space
	query = regexp.MustCompile(`[\t]+`).ReplaceAllString(query, " ")
	query = regexp.MustCompile(`[ ]{2,}`).ReplaceAllString(query, " ")

	// Replace common problematic characters
	// Smart quotes (curly quotes) → regular single quotes
	query = strings.ReplaceAll(query, "\u201c", "'") // Left double quote "
	query = strings.ReplaceAll(query, "\u201d", "'") // Right double quote "
	query = strings.ReplaceAll(query, "\u2018", "'") // Left single quote '
	query = strings.ReplaceAll(query, "\u2019", "'") // Right single quote '

	// Remove zero-width characters and other invisible Unicode
	query = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`).ReplaceAllString(query, "")

	// Normalize line breaks to spaces (DataPrime queries should be single-line for API)
	query = strings.ReplaceAll(query, "\r\n", " ")
	query = strings.ReplaceAll(query, "\r", " ")
	query = strings.ReplaceAll(query, "\n", " ")

	// Trim leading/trailing whitespace
	query = strings.TrimSpace(query)

	return query
}

// applyTierCorrections applies tier-specific corrections to a query.
// Different tiers have different capabilities and requirements.
func applyTierCorrections(query string, tier string) (string, []string) {
	var corrections []string
	corrected := query

	// Archive tier specific corrections are already handled by AutoCorrectDataPrimeQuery
	// (numeric severity → keyword severity)

	// Frequent search tier specific corrections
	// Note: count_distinct → approx_count_distinct is already handled by AutoCorrectDataPrimeQuery
	// since approx_count_distinct is the only variant that works reliably
	_ = tier // Tier-specific corrections may be added in the future

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
		// Check for specific invalid keywords/functions
		if strings.Contains(lowerError, "timestampformat") {
			return `**'timestampformat' is not a valid DataPrime keyword.** Use roundTime() in groupby for time bucketing.

**Correct syntax:**
` + "```" + `
source logs
| filter $m.severity >= ERROR
| groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname
| aggregate count() as error_count
| sortby time_bucket
` + "```" + `

**Time bucket intervals:**
- roundTime($m.timestamp, 5m) - 5 minute buckets
- roundTime($m.timestamp, 1h) - hourly buckets
- roundTime($m.timestamp, 1d) - daily buckets`
		}

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
		// Specific suggestion for bin() function
		if strings.Contains(lowerError, "'bin'") {
			return `**bin() is not a valid DataPrime function.** Use roundTime() instead for time bucketing.

**Correct syntax:**
- roundTime($m.timestamp, 1h) - Group by hour
- roundTime($m.timestamp, 15m) - Group by 15 minutes
- roundTime($m.timestamp, 1d) - Group by day

**Example query:**
source logs | filter $m.severity >= ERROR | groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname aggregate count() as error_count | sortby time_bucket desc`
		}

		return `Unknown function. Common DataPrime functions:

**Aggregation:** count(), sum(), avg(), min(), max(), approx_count_distinct()
**String:** contains(), startsWith(), endsWith(), matches(), toLowerCase(), toUpperCase(), trim(), concat()
**Time:** now(), roundTime(), parseTimestamp(), formatTimestamp(), diffTime()
**Conditional:** if(), case {}, coalesce()

**Time bucketing:** Use roundTime($m.timestamp, <interval>) where interval is: 1s, 5m, 1h, 1d, etc.`
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
