// Package tools provides MCP tool implementations for IBM Cloud Logs.
// This file contains result analysis functions for intelligent summaries.
package tools

import (
	"fmt"
	"strings"
)

// ResultAnalysis provides intelligent analysis of query/list results
type ResultAnalysis struct {
	Summary         string      `json:"summary"`
	Trends          []Trend     `json:"trends,omitempty"`
	Anomalies       []Anomaly   `json:"anomalies,omitempty"`
	Insights        []Insight   `json:"insights,omitempty"`
	Recommendations []string    `json:"recommendations,omitempty"`
	Statistics      *Statistics `json:"statistics,omitempty"`
}

// Trend represents a detected trend in the data
type Trend struct {
	Type        string  `json:"type"`        // increasing, decreasing, stable, volatile
	Metric      string  `json:"metric"`      // what is trending
	Change      float64 `json:"change"`      // percentage change
	Description string  `json:"description"` // human-readable description
}

// Anomaly represents an unusual pattern in the data
type Anomaly struct {
	Type        string `json:"type"`        // spike, gap, unusual_value
	Description string `json:"description"` // what was detected
	Severity    string `json:"severity"`    // info, warning, critical
	Location    string `json:"location"`    // where in the data
}

// Insight represents an actionable insight extracted from the data
type Insight struct {
	Category    string `json:"category"`    // error, performance, security, usage
	Title       string `json:"title"`       // short title
	Description string `json:"description"` // detailed description
	Action      string `json:"action"`      // suggested action
}

// Statistics provides statistical summary of the data
type Statistics struct {
	TotalRecords int                   `json:"total_records"`
	TimeSpan     string                `json:"time_span,omitempty"`
	ErrorRate    float64               `json:"error_rate,omitempty"`
	TopValues    map[string][]TopValue `json:"top_values,omitempty"`
}

// TopValue represents a frequently occurring value
type TopValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// AnalyzeQueryResults performs intelligent analysis on query results
func AnalyzeQueryResults(result map[string]interface{}) *ResultAnalysis {
	analysis := &ResultAnalysis{
		Trends:          []Trend{},
		Anomalies:       []Anomaly{},
		Insights:        []Insight{},
		Recommendations: []string{},
	}

	events, ok := result["events"].([]interface{})
	if !ok || len(events) == 0 {
		analysis.Summary = "No events found in the query results."
		analysis.Recommendations = append(analysis.Recommendations,
			"Try expanding the time range",
			"Check if the filter conditions are too restrictive",
			"Verify the application/subsystem names are correct",
		)
		return analysis
	}

	// Build statistics
	analysis.Statistics = buildStatistics(events)

	// Detect severity distribution and error rate
	severityDist := analyzeSeverityDistribution(events)
	errorRate := calculateErrorRate(severityDist)
	analysis.Statistics.ErrorRate = errorRate

	// Build summary
	analysis.Summary = buildAnalysisSummary(events, severityDist, errorRate)

	// Detect trends
	analysis.Trends = detectTrends(events)

	// Detect anomalies
	analysis.Anomalies = detectAnomalies(events, severityDist)

	// Generate insights
	analysis.Insights = generateInsights(events, severityDist, errorRate)

	// Generate recommendations
	analysis.Recommendations = generateRecommendations(events, severityDist, errorRate)

	return analysis
}

// buildStatistics builds statistical summary from events
func buildStatistics(events []interface{}) *Statistics {
	stats := &Statistics{
		TotalRecords: len(events),
		TopValues:    make(map[string][]TopValue),
	}

	// Extract time span
	timeRange := extractTimeRange(events)
	if timeRange != "" {
		stats.TimeSpan = timeRange
	}

	// Extract top applications
	topApps := extractTopValues(events, "applicationname", 5)
	if len(topApps) > 0 {
		for _, vc := range topApps {
			stats.TopValues["applications"] = append(stats.TopValues["applications"], TopValue(vc))
		}
	}

	// Extract top subsystems
	topSubs := extractTopValues(events, "subsystemname", 5)
	if len(topSubs) > 0 {
		for _, vc := range topSubs {
			stats.TopValues["subsystems"] = append(stats.TopValues["subsystems"], TopValue(vc))
		}
	}

	return stats
}

// calculateErrorRate calculates the error rate from severity distribution
func calculateErrorRate(severityDist map[string]int) float64 {
	total := 0
	errors := 0
	for sev, count := range severityDist {
		total += count
		if sev == "Error" || sev == "Critical" {
			errors += count
		}
	}
	if total == 0 {
		return 0
	}
	return float64(errors) * 100.0 / float64(total)
}

// buildAnalysisSummary builds a human-readable summary
func buildAnalysisSummary(events []interface{}, severityDist map[string]int, errorRate float64) string {
	var parts []string

	// Count summary
	parts = append(parts, fmt.Sprintf("Found %d log entries.", len(events)))

	// Error rate summary
	if errorRate > 0 {
		if errorRate > 10 {
			parts = append(parts, fmt.Sprintf("⚠️ High error rate: %.1f%% of logs are errors or critical.", errorRate))
		} else if errorRate > 1 {
			parts = append(parts, fmt.Sprintf("Error rate: %.1f%%", errorRate))
		}
	}

	// Severity breakdown
	if len(severityDist) > 0 {
		var sevParts []string
		for sev, count := range severityDist {
			sevParts = append(sevParts, fmt.Sprintf("%s: %d", sev, count))
		}
		parts = append(parts, fmt.Sprintf("Severity distribution: %s", strings.Join(sevParts, ", ")))
	}

	return strings.Join(parts, " ")
}

// detectTrends analyzes events for trends over time
func detectTrends(events []interface{}) []Trend {
	var trends []Trend

	if len(events) < 10 {
		return trends // Not enough data for trend analysis
	}

	// Split events into first and second half by time
	midpoint := len(events) / 2
	firstHalf := events[:midpoint]
	secondHalf := events[midpoint:]

	// Count errors in each half
	firstErrors := countSeverityAbove(firstHalf, 5)
	secondErrors := countSeverityAbove(secondHalf, 5)

	if firstErrors > 0 || secondErrors > 0 {
		change := float64(secondErrors-firstErrors) / float64(max(firstErrors, 1)) * 100

		if change > 50 {
			trends = append(trends, Trend{
				Type:        "increasing",
				Metric:      "error_rate",
				Change:      change,
				Description: fmt.Sprintf("Error rate is increasing: %.0f%% more errors in recent logs", change),
			})
		} else if change < -50 {
			trends = append(trends, Trend{
				Type:        "decreasing",
				Metric:      "error_rate",
				Change:      change,
				Description: fmt.Sprintf("Error rate is decreasing: %.0f%% fewer errors in recent logs", -change),
			})
		}
	}

	return trends
}

// countSeverityAbove counts events with severity >= threshold
func countSeverityAbove(events []interface{}, threshold int) int {
	count := 0
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			var severity int
			if sev, ok := eventMap["severity"].(float64); ok {
				severity = int(sev)
			} else if metadata, ok := eventMap["metadata"].(map[string]interface{}); ok {
				if sev, ok := metadata["severity"].(float64); ok {
					severity = int(sev)
				}
			}
			if severity >= threshold {
				count++
			}
		}
	}
	return count
}

// detectAnomalies looks for unusual patterns in the data
func detectAnomalies(events []interface{}, severityDist map[string]int) []Anomaly {
	var anomalies []Anomaly

	// Check for critical errors
	if critical, ok := severityDist["Critical"]; ok && critical > 0 {
		anomalies = append(anomalies, Anomaly{
			Type:        "critical_errors",
			Description: fmt.Sprintf("Found %d critical severity logs", critical),
			Severity:    "critical",
			Location:    "severity distribution",
		})
	}

	// Check for concentration of errors from single source
	topApps := extractTopValues(events, "applicationname", 1)
	if len(topApps) > 0 && len(events) > 10 {
		topAppPercent := float64(topApps[0].Count) * 100.0 / float64(len(events))
		if topAppPercent > 80 {
			anomalies = append(anomalies, Anomaly{
				Type:        "concentration",
				Description: fmt.Sprintf("%.0f%% of logs are from '%s' - potential issue with this service", topAppPercent, topApps[0].Value),
				Severity:    "warning",
				Location:    "application distribution",
			})
		}
	}

	return anomalies
}

// generateInsights extracts actionable insights from the data
func generateInsights(events []interface{}, severityDist map[string]int, errorRate float64) []Insight {
	var insights []Insight

	// High error rate insight
	if errorRate > 5 {
		insights = append(insights, Insight{
			Category:    "error",
			Title:       "High Error Rate Detected",
			Description: fmt.Sprintf("%.1f%% of logs are errors or critical severity", errorRate),
			Action:      "Create an alert to monitor this condition and investigate the root cause",
		})
	}

	// Multiple critical errors
	if critical, ok := severityDist["Critical"]; ok && critical > 5 {
		insights = append(insights, Insight{
			Category:    "error",
			Title:       "Multiple Critical Errors",
			Description: fmt.Sprintf("%d critical severity logs detected", critical),
			Action:      "Immediate investigation recommended - check for service outages or data issues",
		})
	}

	// Large result set insight
	if len(events) >= MaxSSEEvents {
		insights = append(insights, Insight{
			Category:    "usage",
			Title:       "Large Result Set",
			Description: fmt.Sprintf("Query returned maximum %d events - there may be more data", MaxSSEEvents),
			Action:      "Use time-based pagination or add filters to get complete results",
		})
	}

	return insights
}

// generateRecommendations creates actionable recommendations
func generateRecommendations(events []interface{}, severityDist map[string]int, errorRate float64) []string {
	var recs []string

	// Error rate recommendations
	if errorRate > 10 {
		recs = append(recs,
			"Consider creating an alert with: create_alert for this error pattern",
			"Use get_query_templates with name='error_details' for deeper investigation",
		)
	}

	// Critical errors
	if critical, ok := severityDist["Critical"]; ok && critical > 0 {
		recs = append(recs,
			"Investigate critical errors immediately - they often indicate service disruption",
		)
	}

	// Dashboard recommendation
	if len(events) > 50 {
		recs = append(recs,
			"Create a dashboard with: create_dashboard to visualize this data over time",
		)
	}

	// If mostly healthy, suggest monitoring setup
	if errorRate < 1 && len(events) > 10 {
		recs = append(recs,
			"System appears healthy - consider setting up proactive alerting with: suggest_alert",
		)
	}

	return recs
}

// AnalyzeResourceList provides analysis for list operations (dashboards, alerts, etc.)
func AnalyzeResourceList(items []interface{}, resourceType string) *ResultAnalysis {
	analysis := &ResultAnalysis{
		Statistics: &Statistics{
			TotalRecords: len(items),
		},
	}

	if len(items) == 0 {
		analysis.Summary = fmt.Sprintf("No %s found.", resourceType)
		analysis.Recommendations = []string{
			fmt.Sprintf("Create a new %s with: create_%s", resourceType, resourceType),
		}
		return analysis
	}

	analysis.Summary = fmt.Sprintf("Found %d %s(s).", len(items), resourceType)

	// Extract names for quick reference
	names := extractFieldValues(items, []string{"name", "title"}, 5)
	if len(names) > 0 {
		analysis.Insights = append(analysis.Insights, Insight{
			Category:    "info",
			Title:       fmt.Sprintf("Top %ss", resourceType),
			Description: strings.Join(names, ", "),
		})
	}

	return analysis
}

// FormatAnalysisAsMarkdown formats analysis results as markdown
func FormatAnalysisAsMarkdown(analysis *ResultAnalysis) string {
	if analysis == nil {
		return ""
	}

	var parts []string

	// Summary
	if analysis.Summary != "" {
		parts = append(parts, fmt.Sprintf("## Analysis Summary\n%s", analysis.Summary))
	}

	// Statistics
	if analysis.Statistics != nil {
		stats := analysis.Statistics
		statsParts := []string{fmt.Sprintf("- **Total Records:** %d", stats.TotalRecords)}
		if stats.TimeSpan != "" {
			statsParts = append(statsParts, fmt.Sprintf("- **Time Span:** %s", stats.TimeSpan))
		}
		if stats.ErrorRate > 0 {
			statsParts = append(statsParts, fmt.Sprintf("- **Error Rate:** %.1f%%", stats.ErrorRate))
		}
		parts = append(parts, "### Statistics\n"+strings.Join(statsParts, "\n"))
	}

	// Anomalies
	if len(analysis.Anomalies) > 0 {
		var anomalyParts []string
		for _, a := range analysis.Anomalies {
			icon := "ℹ️"
			switch a.Severity {
			case "warning":
				icon = "⚠️"
			case "critical":
				icon = "🚨"
			}
			anomalyParts = append(anomalyParts, fmt.Sprintf("- %s **%s:** %s", icon, a.Type, a.Description))
		}
		parts = append(parts, "### Anomalies Detected\n"+strings.Join(anomalyParts, "\n"))
	}

	// Trends
	if len(analysis.Trends) > 0 {
		var trendParts []string
		for _, t := range analysis.Trends {
			icon := "📈"
			if t.Type == "decreasing" {
				icon = "📉"
			}
			trendParts = append(trendParts, fmt.Sprintf("- %s %s", icon, t.Description))
		}
		parts = append(parts, "### Trends\n"+strings.Join(trendParts, "\n"))
	}

	// Insights
	if len(analysis.Insights) > 0 {
		var insightParts []string
		for _, i := range analysis.Insights {
			insightParts = append(insightParts, fmt.Sprintf("- **%s:** %s\n  → Action: %s", i.Title, i.Description, i.Action))
		}
		parts = append(parts, "### Insights\n"+strings.Join(insightParts, "\n"))
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		var recParts []string
		for _, r := range analysis.Recommendations {
			recParts = append(recParts, fmt.Sprintf("- %s", r))
		}
		parts = append(parts, "### Recommendations\n"+strings.Join(recParts, "\n"))
	}

	return strings.Join(parts, "\n\n")
}
