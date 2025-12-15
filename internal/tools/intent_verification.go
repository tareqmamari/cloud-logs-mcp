// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file implements intent verification based on research showing
// self-verification improves accuracy by 10-20% (ReAct + CoT-SC pattern).
package tools

import (
	"strings"
)

// IntentVerification provides self-verification for parsed intents.
// Based on research showing clarifying questions improve accuracy.
type IntentVerification struct {
	OriginalIntent      string          `json:"original_intent"`
	ParsedIntent        string          `json:"parsed_intent"`
	Confidence          float64         `json:"confidence"`
	IntentType          IntentType      `json:"intent_type"`
	ExtractedEntities   *IntentEntities `json:"extracted_entities,omitempty"`
	ClarifyingQuestions []string        `json:"clarifying_questions,omitempty"`
	AlternativeIntents  []string        `json:"alternative_intents,omitempty"`
	Ambiguities         []string        `json:"ambiguities,omitempty"`
}

// IntentType categorizes the type of user intent
type IntentType string

// IntentType constants for categorizing user intents
const (
	IntentQuery       IntentType = "query"       // User wants to search/query logs
	IntentInvestigate IntentType = "investigate" // User wants to investigate an issue
	IntentMonitor     IntentType = "monitor"     // User wants to set up monitoring
	IntentVisualize   IntentType = "visualize"   // User wants dashboards/charts
	IntentConfigure   IntentType = "configure"   // User wants to configure settings
	IntentLearn       IntentType = "learn"       // User wants to learn/understand
	IntentExplore     IntentType = "explore"     // User wants to explore capabilities
	IntentUnknown     IntentType = "unknown"     // Intent couldn't be determined
)

// IntentEntities holds entities extracted from the intent
type IntentEntities struct {
	Services  []string `json:"services,omitempty"`   // Service/app names mentioned
	TimeRange string   `json:"time_range,omitempty"` // Time range mentioned
	Severity  string   `json:"severity,omitempty"`   // Severity level mentioned
	Keywords  []string `json:"keywords,omitempty"`   // Important keywords
	TraceID   string   `json:"trace_id,omitempty"`   // Trace ID if mentioned
	ErrorType string   `json:"error_type,omitempty"` // Type of error mentioned
}

// VerifyIntent performs self-verification on a parsed intent.
// Returns verification result with confidence and potential clarifications.
func VerifyIntent(rawIntent string) *IntentVerification {
	verification := &IntentVerification{
		OriginalIntent: rawIntent,
		Confidence:     0.0,
		IntentType:     IntentUnknown,
	}

	if rawIntent == "" {
		verification.ClarifyingQuestions = []string{
			"What would you like to do with your logs?",
			"Are you investigating an issue, setting up monitoring, or exploring data?",
		}
		return verification
	}

	intentLower := strings.ToLower(rawIntent)

	// Extract entities
	verification.ExtractedEntities = extractEntities(intentLower)

	// Determine intent type and confidence
	verification.IntentType, verification.Confidence = classifyIntent(intentLower)
	verification.ParsedIntent = generateParsedIntent(verification.IntentType, verification.ExtractedEntities)

	// Check for ambiguities
	verification.Ambiguities = detectAmbiguities(intentLower, verification.IntentType)

	// Generate clarifying questions if confidence is low
	if verification.Confidence < 0.7 || len(verification.Ambiguities) > 0 {
		verification.ClarifyingQuestions = generateClarifyingQuestions(verification)
	}

	// Suggest alternative interpretations
	if verification.Confidence < 0.9 {
		verification.AlternativeIntents = generateAlternatives(intentLower, verification.IntentType)
	}

	return verification
}

// extractEntities extracts named entities from the intent
func extractEntities(intent string) *IntentEntities {
	entities := &IntentEntities{}

	// Extract service names (common patterns)
	servicePatterns := []string{"-api", "-service", "-gateway", "-worker", "-app"}
	words := strings.Fields(intent)
	for _, word := range words {
		for _, pattern := range servicePatterns {
			if strings.Contains(word, pattern) {
				entities.Services = append(entities.Services, word)
				break
			}
		}
	}

	// Extract time ranges
	timePatterns := map[string]string{
		"last hour":     "1h",
		"last 24 hours": "24h",
		"last day":      "24h",
		"last week":     "7d",
		"today":         "today",
		"yesterday":     "yesterday",
		"this morning":  "morning",
	}
	for pattern, value := range timePatterns {
		if strings.Contains(intent, pattern) {
			entities.TimeRange = value
			break
		}
	}

	// Extract severity
	severityPatterns := map[string]string{
		"error":    "ERROR",
		"critical": "CRITICAL",
		"warning":  "WARNING",
		"warn":     "WARNING",
		"info":     "INFO",
		"debug":    "DEBUG",
	}
	for pattern, value := range severityPatterns {
		if strings.Contains(intent, pattern) {
			entities.Severity = value
			break
		}
	}

	// Extract error types
	errorPatterns := []string{"timeout", "connection", "authentication", "authorization", "500", "404", "null pointer", "exception", "crash"}
	for _, pattern := range errorPatterns {
		if strings.Contains(intent, pattern) {
			entities.ErrorType = pattern
			break
		}
	}

	// Extract trace ID patterns (hex strings of 16+ chars)
	for _, word := range words {
		if len(word) >= 16 && isHexString(word) {
			entities.TraceID = word
			break
		}
	}

	// Extract keywords
	importantKeywords := []string{"spike", "increase", "decrease", "pattern", "anomaly", "root cause", "failing", "slow", "high", "low"}
	for _, kw := range importantKeywords {
		if strings.Contains(intent, kw) {
			entities.Keywords = append(entities.Keywords, kw)
		}
	}

	return entities
}

// classifyIntent determines the intent type and confidence
func classifyIntent(intent string) (IntentType, float64) {
	// Scoring system for each intent type
	scores := map[IntentType]float64{
		IntentQuery:       0.0,
		IntentInvestigate: 0.0,
		IntentMonitor:     0.0,
		IntentVisualize:   0.0,
		IntentConfigure:   0.0,
		IntentLearn:       0.0,
		IntentExplore:     0.0,
	}

	// Query indicators
	queryWords := []string{"search", "find", "query", "show", "get", "list", "filter", "look for"}
	for _, word := range queryWords {
		if strings.Contains(intent, word) {
			scores[IntentQuery] += 0.3
		}
	}

	// Investigation indicators
	investigateWords := []string{"investigate", "debug", "troubleshoot", "root cause", "why", "what happened", "issue", "problem", "failing", "broken", "error", "incident"}
	for _, word := range investigateWords {
		if strings.Contains(intent, word) {
			scores[IntentInvestigate] += 0.3
		}
	}

	// Monitor indicators
	monitorWords := []string{"alert", "notify", "monitor", "watch", "track", "threshold", "alarm"}
	for _, word := range monitorWords {
		if strings.Contains(intent, word) {
			scores[IntentMonitor] += 0.3
		}
	}

	// Visualize indicators
	visualizeWords := []string{"dashboard", "chart", "graph", "visualize", "display", "report"}
	for _, word := range visualizeWords {
		if strings.Contains(intent, word) {
			scores[IntentVisualize] += 0.3
		}
	}

	// Configure indicators
	configureWords := []string{"create", "setup", "configure", "set up", "enable", "disable", "update", "delete", "policy", "retention"}
	for _, word := range configureWords {
		if strings.Contains(intent, word) {
			scores[IntentConfigure] += 0.25
		}
	}

	// Learn indicators
	learnWords := []string{"how", "what is", "explain", "help", "learn", "understand", "syntax", "example", "tutorial"}
	for _, word := range learnWords {
		if strings.Contains(intent, word) {
			scores[IntentLearn] += 0.3
		}
	}

	// Explore indicators
	exploreWords := []string{"what can", "available", "capabilities", "tools", "options", "features"}
	for _, word := range exploreWords {
		if strings.Contains(intent, word) {
			scores[IntentExplore] += 0.3
		}
	}

	// Find the highest scoring intent
	maxScore := 0.0
	bestIntent := IntentUnknown
	for intent, score := range scores {
		if score > maxScore {
			maxScore = score
			bestIntent = intent
		}
	}

	// Normalize confidence (cap at 1.0)
	confidence := maxScore
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Boost confidence if there's a clear winner
	secondBest := 0.0
	for intent, score := range scores {
		if intent != bestIntent && score > secondBest {
			secondBest = score
		}
	}
	if maxScore > 0 && secondBest > 0 {
		// Clear distinction boosts confidence
		if maxScore > secondBest*1.5 {
			confidence = min(confidence+0.2, 1.0)
		}
	}

	return bestIntent, confidence
}

// generateParsedIntent creates a normalized intent description
func generateParsedIntent(intentType IntentType, entities *IntentEntities) string {
	parts := []string{}

	switch intentType {
	case IntentQuery:
		parts = append(parts, "Search logs")
	case IntentInvestigate:
		parts = append(parts, "Investigate issue")
	case IntentMonitor:
		parts = append(parts, "Set up monitoring")
	case IntentVisualize:
		parts = append(parts, "Create visualization")
	case IntentConfigure:
		parts = append(parts, "Configure settings")
	case IntentLearn:
		parts = append(parts, "Learn about")
	case IntentExplore:
		parts = append(parts, "Explore capabilities")
	default:
		parts = append(parts, "Unknown action")
	}

	if entities != nil {
		if len(entities.Services) > 0 {
			parts = append(parts, "for "+strings.Join(entities.Services, ", "))
		}
		if entities.Severity != "" {
			parts = append(parts, "with severity "+entities.Severity)
		}
		if entities.TimeRange != "" {
			parts = append(parts, "in "+entities.TimeRange)
		}
		if entities.ErrorType != "" {
			parts = append(parts, "related to "+entities.ErrorType)
		}
	}

	return strings.Join(parts, " ")
}

// detectAmbiguities finds potential ambiguities in the intent
func detectAmbiguities(intent string, _ IntentType) []string {
	ambiguities := []string{}

	// Check for competing intents
	hasQuery := strings.Contains(intent, "search") || strings.Contains(intent, "find")
	hasMonitor := strings.Contains(intent, "alert") || strings.Contains(intent, "monitor")
	hasVisualize := strings.Contains(intent, "dashboard") || strings.Contains(intent, "chart")

	if hasQuery && hasMonitor {
		ambiguities = append(ambiguities, "Intent could be searching logs OR setting up alerts")
	}
	if hasQuery && hasVisualize {
		ambiguities = append(ambiguities, "Intent could be querying data OR creating visualizations")
	}

	// Check for missing context
	if !strings.Contains(intent, "last") && !strings.Contains(intent, "hour") &&
		!strings.Contains(intent, "day") && !strings.Contains(intent, "today") {
		ambiguities = append(ambiguities, "No time range specified")
	}

	// Check for vague service references
	if strings.Contains(intent, "my service") || strings.Contains(intent, "the app") ||
		strings.Contains(intent, "our system") {
		ambiguities = append(ambiguities, "Service name is not specific")
	}

	return ambiguities
}

// generateClarifyingQuestions creates questions to resolve ambiguities
func generateClarifyingQuestions(v *IntentVerification) []string {
	questions := []string{}

	// Add questions based on ambiguities
	for _, amb := range v.Ambiguities {
		switch {
		case strings.Contains(amb, "searching logs OR setting up alerts"):
			questions = append(questions, "Do you want to search existing logs, or create an alert for future events?")
		case strings.Contains(amb, "querying data OR creating visualizations"):
			questions = append(questions, "Do you need a one-time query result, or a persistent dashboard?")
		case strings.Contains(amb, "No time range"):
			questions = append(questions, "What time range should I search? (e.g., last hour, last 24 hours, yesterday)")
		case strings.Contains(amb, "Service name"):
			questions = append(questions, "Which specific service or application are you interested in?")
		}
	}

	// Add questions based on missing entities
	if v.ExtractedEntities != nil {
		if len(v.ExtractedEntities.Services) == 0 && v.IntentType == IntentInvestigate {
			questions = append(questions, "Which service or application is experiencing the issue?")
		}
		if v.ExtractedEntities.Severity == "" && (v.IntentType == IntentQuery || v.IntentType == IntentInvestigate) {
			questions = append(questions, "Should I focus on errors only, or include warnings and info logs?")
		}
	}

	// Limit to 3 questions
	if len(questions) > 3 {
		questions = questions[:3]
	}

	return questions
}

// generateAlternatives suggests alternative interpretations
func generateAlternatives(_ string, primaryType IntentType) []string {
	alternatives := []string{}

	switch primaryType {
	case IntentQuery:
		alternatives = append(alternatives, "Investigate errors in logs")
		alternatives = append(alternatives, "Set up alert for matching logs")
	case IntentInvestigate:
		alternatives = append(alternatives, "Just search logs without analysis")
		alternatives = append(alternatives, "Create dashboard to visualize the issue")
	case IntentMonitor:
		alternatives = append(alternatives, "Query logs first to validate the pattern")
		alternatives = append(alternatives, "Create a dashboard instead of an alert")
	case IntentVisualize:
		alternatives = append(alternatives, "Get raw query results")
		alternatives = append(alternatives, "Set up alerting based on the metrics")
	}

	return alternatives
}

// isHexString checks if a string contains only hex characters
func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}
