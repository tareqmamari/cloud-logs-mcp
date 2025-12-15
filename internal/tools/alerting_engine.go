// Package tools provides MCP tools for IBM Cloud Logs operations.
package tools

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// AlertingMethodology represents the alerting approach for a component type
type AlertingMethodology string

const (
	// MethodologyRED uses Rate, Errors, Duration metrics (for services)
	MethodologyRED AlertingMethodology = "RED"
	// MethodologyUSE uses Utilization, Saturation, Errors metrics (for resources)
	MethodologyUSE AlertingMethodology = "USE"
	// MethodologyGoldenSignals uses the four golden signals
	MethodologyGoldenSignals AlertingMethodology = "GOLDEN_SIGNALS"
)

// ComponentType represents the type of system component being monitored
type ComponentType string

// Component type constants for categorizing system components
const (
	ComponentWebService   ComponentType = "web_service"
	ComponentAPIGateway   ComponentType = "api_gateway"
	ComponentDatabase     ComponentType = "database"
	ComponentCache        ComponentType = "cache"
	ComponentMessageQueue ComponentType = "message_queue"
	ComponentLoadBalancer ComponentType = "load_balancer"
	ComponentWorker       ComponentType = "worker"
	ComponentCronJob      ComponentType = "cron_job"
	ComponentStorage      ComponentType = "storage"
	ComponentNetwork      ComponentType = "network"
	ComponentKubernetes   ComponentType = "kubernetes"
	ComponentServerless   ComponentType = "serverless"
	ComponentMicroservice ComponentType = "microservice"
	ComponentMonolith     ComponentType = "monolith"
	ComponentCustom       ComponentType = "custom"
)

// SeverityLevel represents alert severity for incident classification
type SeverityLevel string

// Severity level constants for incident classification
const (
	SeverityP1Critical SeverityLevel = "P1" // User-facing + high burn rate (page)
	SeverityP2Warning  SeverityLevel = "P2" // Saturation impending / low burn rate (ticket)
	SeverityP3Info     SeverityLevel = "P3" // Informational / trend detection
)

// BurnRateWindow represents a time window for burn rate calculation
type BurnRateWindow struct {
	Duration  time.Duration `json:"duration"`
	BurnRate  float64       `json:"burn_rate"` // Multiplier of sustainable burn rate
	Severity  SeverityLevel `json:"severity"`
	AlertType string        `json:"alert_type"` // "fast_burn" or "slow_burn"
}

// SLOConfig represents a Service Level Objective configuration
type SLOConfig struct {
	Target             float64       `json:"target"`               // e.g., 0.999 for 99.9%
	Window             time.Duration `json:"window"`               // e.g., 30 days
	ErrorBudget        float64       `json:"error_budget"`         // Calculated: 1 - Target
	MonthlyErrorBudget float64       `json:"monthly_error_budget"` // In hours or percentage
}

// BurnRateConfig represents multi-window burn rate alerting configuration
type BurnRateConfig struct {
	FastBurnWindows []BurnRateWindow `json:"fast_burn_windows"`
	SlowBurnWindows []BurnRateWindow `json:"slow_burn_windows"`
	SLO             SLOConfig        `json:"slo"`
}

// AlertStrategyConfig defines the complete alerting strategy for a component
type AlertStrategyConfig struct {
	ComponentType      ComponentType           `json:"component_type"`
	Methodology        AlertingMethodology     `json:"methodology"`
	RecommendedMetrics []MetricRecommendation  `json:"recommended_metrics"`
	BurnRateConfig     *BurnRateConfig         `json:"burn_rate_config,omitempty"`
	DynamicBaselines   []DynamicBaselineConfig `json:"dynamic_baselines,omitempty"`
	RunbookTemplate    string                  `json:"runbook_template"`
	Labels             map[string]string       `json:"labels"`
}

// MetricRecommendation represents a recommended metric to monitor
type MetricRecommendation struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`   // "counter", "gauge", "histogram"
	Signal           string   `json:"signal"` // "rate", "errors", "duration", "utilization", "saturation"
	Query            string   `json:"query"`  // DataPrime query template
	DefaultThreshold float64  `json:"default_threshold"`
	Unit             string   `json:"unit"`
	Description      string   `json:"description"`
	BestPractices    []string `json:"best_practices"`
}

// DynamicBaselineConfig represents configuration for dynamic threshold detection
type DynamicBaselineConfig struct {
	MetricName       string        `json:"metric_name"`
	SeasonalityType  string        `json:"seasonality_type"` // "hourly", "daily", "weekly"
	StdDevMultiplier float64       `json:"std_dev_multiplier"`
	MinDataPoints    int           `json:"min_data_points"`
	LookbackWindow   time.Duration `json:"lookback_window"`
}

// AdvancedAlertSuggestion represents a comprehensive alert recommendation
type AdvancedAlertSuggestion struct {
	// Basic Info
	Name        string `json:"name"`
	Description string `json:"description"`

	// Classification
	Severity    SeverityLevel       `json:"severity"`
	Methodology AlertingMethodology `json:"methodology"`
	Signal      string              `json:"signal"`

	// Query & Condition
	Query             string             `json:"query"`
	Condition         AlertCondition     `json:"condition"`
	BurnRateCondition *BurnRateCondition `json:"burn_rate_condition,omitempty"`

	// Multi-window config
	Windows []AlertWindow `json:"windows,omitempty"`

	// Actionability (REQUIRED)
	RunbookURL       string   `json:"runbook_url"`
	SuggestedActions []string `json:"suggested_actions"`

	// Metadata
	Labels   map[string]string `json:"labels"`
	Schedule AlertSchedule     `json:"schedule"`

	// Infrastructure as Code
	TerraformConfig string `json:"terraform_config,omitempty"`

	// Explanation
	Explanation   string   `json:"explanation"`
	BestPractices []string `json:"best_practices"`
	References    []string `json:"references"`
}

// BurnRateCondition represents SLO-based burn rate alerting
type BurnRateCondition struct {
	SLOTarget          float64 `json:"slo_target"`
	ErrorBudgetPercent float64 `json:"error_budget_percent"`
	BurnRate           float64 `json:"burn_rate"`
	WindowDuration     string  `json:"window_duration"`
	ConsumptionPercent float64 `json:"consumption_percent"`
}

// AlertWindow represents a single window in multi-window alerting
type AlertWindow struct {
	Duration string  `json:"duration"`
	BurnRate float64 `json:"burn_rate"`
	Type     string  `json:"type"` // "short" or "long"
}

// AlertingStrategyMatrix is the central registry mapping component types to alerting strategies
var AlertingStrategyMatrix = map[ComponentType]AlertStrategyConfig{
	ComponentWebService: {
		ComponentType: ComponentWebService,
		Methodology:   MethodologyRED,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "request_rate",
				Type:             "counter",
				Signal:           "rate",
				Query:            `source logs | filter $d.type == 'http_request' | stats count() as requests by bin(1m)`,
				DefaultThreshold: 0,
				Unit:             "requests/second",
				Description:      "Request rate indicates service load and can detect traffic anomalies",
				BestPractices: []string{
					"Set baseline from historical p50 traffic",
					"Alert on both high AND low traffic (traffic drop may indicate upstream issues)",
					"Use dynamic baselines for services with variable traffic patterns",
				},
			},
			{
				Name:             "error_rate",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $m.severity >= 5 OR $d.status_code >= 500 | stats count() as errors by bin(1m)`,
				DefaultThreshold: 0.01, // 1% error rate
				Unit:             "percentage",
				Description:      "Error rate as percentage of total requests - primary SLI for availability SLO",
				BestPractices: []string{
					"Use burn rate alerting instead of static thresholds",
					"Alert on error budget consumption rate, not absolute errors",
					"Differentiate between client errors (4xx) and server errors (5xx)",
				},
			},
			{
				Name:             "latency_p99",
				Type:             "histogram",
				Signal:           "duration",
				Query:            `source logs | filter $d.response_time_ms exists | stats percentile($d.response_time_ms, 99) as p99_latency by bin(5m)`,
				DefaultThreshold: 500, // 500ms
				Unit:             "milliseconds",
				Description:      "P99 latency represents tail latency experienced by 1% of users",
				BestPractices: []string{
					"Alert on P99, not average latency",
					"Set thresholds based on SLO (e.g., 'P99 < 200ms for 99.9% of requests')",
					"Consider separate alerts for different endpoints with different latency budgets",
				},
			},
		},
		RunbookTemplate: "## Web Service Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check service health endpoint: GET /health\n" +
			"2. Review recent deployments in last 24 hours\n" +
			"3. Check dependent service health\n\n" +
			"### Investigation Steps\n" +
			"1. Query error logs: source logs | filter $m.severity >= 5 | top 10 $d.error_type\n" +
			"2. Check latency distribution: source logs | stats percentile($d.response_time_ms, 50, 90, 99)\n" +
			"3. Identify affected endpoints: source logs | filter $d.status_code >= 500 | top 10 $d.path\n\n" +
			"### Escalation\n" +
			"- P1: Page on-call engineer immediately\n" +
			"- P2: Create ticket, notify team channel",
		Labels: map[string]string{
			"methodology": "RED",
			"tier":        "service",
		},
	},

	ComponentAPIGateway: {
		ComponentType: ComponentAPIGateway,
		Methodology:   MethodologyRED,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "upstream_error_rate",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.component == 'api_gateway' AND $d.upstream_status >= 500 | stats count() as errors by bin(1m)`,
				DefaultThreshold: 0.005,
				Unit:             "percentage",
				Description:      "Upstream service errors proxied through the gateway",
				BestPractices: []string{
					"Separate upstream errors from gateway errors",
					"Track error rates per upstream service",
					"Alert on circuit breaker activations",
				},
			},
			{
				Name:             "gateway_latency",
				Type:             "histogram",
				Signal:           "duration",
				Query:            `source logs | filter $d.component == 'api_gateway' | stats percentile($d.latency_ms, 99) as p99 by bin(1m)`,
				DefaultThreshold: 100,
				Unit:             "milliseconds",
				Description:      "Gateway processing latency (excluding upstream time)",
				BestPractices: []string{
					"Gateway latency should be minimal (< 50ms typically)",
					"High gateway latency indicates gateway resource constraints",
				},
			},
			{
				Name:             "rate_limit_triggers",
				Type:             "counter",
				Signal:           "saturation",
				Query:            `source logs | filter $d.rate_limited == true | stats count() by bin(5m)`,
				DefaultThreshold: 100,
				Unit:             "count",
				Description:      "Rate limiting activations indicate potential abuse or capacity issues",
				BestPractices: []string{
					"Track rate limits per API key/client",
					"Alert on sudden spikes in rate limiting",
				},
			},
		},
		RunbookTemplate: "## API Gateway Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check gateway health metrics\n" +
			"2. Verify upstream service connectivity\n" +
			"3. Review rate limiting configuration\n\n" +
			"### Investigation Steps\n" +
			"1. Identify affected routes: source logs | filter $d.component == 'api_gateway' | top 10 $d.route\n" +
			"2. Check client distribution: source logs | filter $d.component == 'api_gateway' | top 10 $d.client_id\n\n" +
			"### Escalation\n" +
			"- P1: Page platform team\n" +
			"- P2: Create ticket for API team",
		Labels: map[string]string{
			"methodology": "RED",
			"tier":        "infrastructure",
		},
	},

	ComponentDatabase: {
		ComponentType: ComponentDatabase,
		Methodology:   MethodologyUSE,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "connection_utilization",
				Type:             "gauge",
				Signal:           "utilization",
				Query:            `source logs | filter $d.component == 'database' | stats avg($d.connections_active / $d.connections_max * 100) as utilization by bin(1m)`,
				DefaultThreshold: 80,
				Unit:             "percentage",
				Description:      "Connection pool utilization - high values indicate capacity constraints",
				BestPractices: []string{
					"Alert at 80% utilization (warning) and 95% (critical)",
					"Track connection pool exhaustion events separately",
					"Consider connection pooling or read replicas if consistently high",
				},
			},
			{
				Name:             "query_queue_depth",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.component == 'database' | stats max($d.query_queue_length) as queue_depth by bin(1m)`,
				DefaultThreshold: 10,
				Unit:             "queries",
				Description:      "Query queue depth indicates database saturation",
				BestPractices: []string{
					"Queue depth > 0 indicates queries are waiting",
					"Sustained queueing indicates need for optimization or scaling",
				},
			},
			{
				Name:             "replication_lag",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.component == 'database' AND $d.role == 'replica' | stats max($d.replication_lag_seconds) as lag by bin(1m)`,
				DefaultThreshold: 5,
				Unit:             "seconds",
				Description:      "Replication lag affects read consistency and failover capability",
				BestPractices: []string{
					"Alert if lag exceeds your consistency requirements",
					"High lag during failover increases data loss risk",
				},
			},
			{
				Name:             "slow_queries",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.component == 'database' AND $d.query_duration_ms > 1000 | stats count() as slow_queries by bin(5m)`,
				DefaultThreshold: 10,
				Unit:             "count",
				Description:      "Slow queries impact application performance and may indicate missing indexes",
				BestPractices: []string{
					"Capture query fingerprints for slow query analysis",
					"Track slow query rate as percentage of total queries",
				},
			},
		},
		RunbookTemplate: "## Database Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check database server resource utilization (CPU, Memory, Disk I/O)\n" +
			"2. Review active connections and running queries\n" +
			"3. Check replication status if applicable\n\n" +
			"### Investigation Steps\n" +
			"1. Identify slow queries: source logs | filter $d.query_duration_ms > 1000 | top 10 $d.query_fingerprint\n" +
			"2. Check connection sources: source logs | filter $d.component == 'database' | top 10 $d.client_host\n" +
			"3. Review lock contention: source logs | filter $d.lock_wait_ms > 0 | stats avg($d.lock_wait_ms)\n\n" +
			"### Escalation\n" +
			"- P1: Page DBA on-call\n" +
			"- P2: Create ticket for database team",
		Labels: map[string]string{
			"methodology": "USE",
			"tier":        "data",
		},
	},

	ComponentCache: {
		ComponentType: ComponentCache,
		Methodology:   MethodologyUSE,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "memory_utilization",
				Type:             "gauge",
				Signal:           "utilization",
				Query:            `source logs | filter $d.component == 'redis' OR $d.component == 'memcached' | stats avg($d.used_memory / $d.max_memory * 100) as utilization by bin(1m)`,
				DefaultThreshold: 85,
				Unit:             "percentage",
				Description:      "Memory utilization affects cache eviction behavior",
				BestPractices: []string{
					"Alert before reaching maxmemory to prevent unexpected evictions",
					"Track eviction rate alongside memory utilization",
				},
			},
			{
				Name:             "hit_rate",
				Type:             "gauge",
				Signal:           "utilization",
				Query:            `source logs | filter $d.component == 'cache' | stats sum($d.hits) / (sum($d.hits) + sum($d.misses)) * 100 as hit_rate by bin(5m)`,
				DefaultThreshold: 90,
				Unit:             "percentage",
				Description:      "Cache hit rate indicates cache effectiveness",
				BestPractices: []string{
					"Low hit rate may indicate cache sizing issues or access pattern changes",
					"Alert on sudden drops in hit rate",
					"Track hit rate per key prefix if possible",
				},
			},
			{
				Name:             "eviction_rate",
				Type:             "counter",
				Signal:           "saturation",
				Query:            `source logs | filter $d.component == 'cache' | stats sum($d.evicted_keys) as evictions by bin(5m)`,
				DefaultThreshold: 100,
				Unit:             "keys/5min",
				Description:      "High eviction rate indicates memory pressure",
				BestPractices: []string{
					"Evictions force cache rebuilding, increasing backend load",
					"Sustained evictions indicate need for larger cache or TTL tuning",
				},
			},
		},
		RunbookTemplate: "## Cache Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check cache server memory utilization\n" +
			"2. Review hit/miss ratio trends\n" +
			"3. Check eviction statistics\n\n" +
			"### Investigation Steps\n" +
			"1. Identify hot keys: source logs | filter $d.component == 'cache' | top 10 $d.key_prefix\n" +
			"2. Check client connections: source logs | filter $d.component == 'cache' | stats count() by $d.client\n\n" +
			"### Escalation\n" +
			"- P1: Page infrastructure on-call\n" +
			"- P2: Create ticket for platform team",
		Labels: map[string]string{
			"methodology": "USE",
			"tier":        "caching",
		},
	},

	ComponentMessageQueue: {
		ComponentType: ComponentMessageQueue,
		Methodology:   MethodologyUSE,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "queue_depth",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.component == 'queue' | stats max($d.queue_depth) as depth by $d.queue_name, bin(1m)`,
				DefaultThreshold: 1000,
				Unit:             "messages",
				Description:      "Queue depth indicates consumer lag - primary saturation signal",
				BestPractices: []string{
					"Set threshold based on acceptable processing delay",
					"Alert on rate of queue depth increase, not just absolute value",
					"Track queue depth per queue/topic",
				},
			},
			{
				Name:             "consumer_lag",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.component == 'kafka' | stats max($d.consumer_lag) as lag by $d.consumer_group, bin(1m)`,
				DefaultThreshold: 10000,
				Unit:             "messages",
				Description:      "Consumer lag in Kafka indicates processing backlog",
				BestPractices: []string{
					"Set lag threshold based on message rate and acceptable delay",
					"Alert on lag increasing over time, not just threshold",
				},
			},
			{
				Name:             "dead_letter_queue",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.queue_name contains 'dlq' OR $d.queue_name contains 'dead' | stats count() by bin(5m)`,
				DefaultThreshold: 1,
				Unit:             "messages",
				Description:      "Dead letter queue messages indicate processing failures",
				BestPractices: []string{
					"DLQ messages should always trigger investigation",
					"Track DLQ rate and implement alerting on any DLQ activity",
				},
			},
			{
				Name:             "publish_errors",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.operation == 'publish' AND $m.severity >= 5 | stats count() by bin(1m)`,
				DefaultThreshold: 5,
				Unit:             "errors/min",
				Description:      "Publish errors indicate producer issues or broker problems",
				BestPractices: []string{
					"Track publish success rate, not just errors",
					"Alert on publish latency spikes as early warning",
				},
			},
		},
		RunbookTemplate: "## Message Queue Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check broker health and connectivity\n" +
			"2. Review consumer group status\n" +
			"3. Check for dead letter queue messages\n\n" +
			"### Investigation Steps\n" +
			"1. Identify affected queues: source logs | filter $d.component == 'queue' | top 10 $d.queue_name by $d.queue_depth\n" +
			"2. Check consumer status: source logs | filter $d.consumer_group exists | stats count() by $d.consumer_group, $d.status\n" +
			"3. Review error patterns: source logs | filter $d.component == 'queue' AND $m.severity >= 5 | top 10 $d.error_type\n\n" +
			"### Escalation\n" +
			"- P1: Page platform on-call (message loss risk)\n" +
			"- P2: Create ticket for application team (processing delay)",
		Labels: map[string]string{
			"methodology": "USE",
			"tier":        "messaging",
		},
	},

	ComponentWorker: {
		ComponentType: ComponentWorker,
		Methodology:   MethodologyRED,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "job_success_rate",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.component == 'worker' | stats sum(case when $d.job_status == 'success' then 1 else 0 end) / count() * 100 as success_rate by bin(5m)`,
				DefaultThreshold: 99,
				Unit:             "percentage",
				Description:      "Job success rate is the primary reliability metric for workers",
				BestPractices: []string{
					"Track success rate per job type",
					"Set different thresholds for critical vs non-critical jobs",
				},
			},
			{
				Name:             "job_duration",
				Type:             "histogram",
				Signal:           "duration",
				Query:            `source logs | filter $d.component == 'worker' AND $d.job_duration_ms exists | stats percentile($d.job_duration_ms, 95) as p95_duration by $d.job_type, bin(5m)`,
				DefaultThreshold: 60000, // 60 seconds
				Unit:             "milliseconds",
				Description:      "Job duration affects throughput and resource utilization",
				BestPractices: []string{
					"Alert on jobs exceeding SLA duration",
					"Track duration trends for capacity planning",
				},
			},
			{
				Name:             "retry_rate",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.component == 'worker' AND $d.retry_count > 0 | stats count() by bin(5m)`,
				DefaultThreshold: 10,
				Unit:             "count",
				Description:      "Retry rate indicates transient failures affecting reliability",
				BestPractices: []string{
					"High retry rates indicate upstream instability",
					"Track jobs exhausting retry budget",
				},
			},
		},
		RunbookTemplate: "## Worker Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check worker process health\n" +
			"2. Review job queue depth\n" +
			"3. Check dependent service connectivity\n\n" +
			"### Investigation Steps\n" +
			"1. Identify failing jobs: source logs | filter $d.component == 'worker' AND $d.job_status == 'failed' | top 10 $d.job_type\n" +
			"2. Check error reasons: source logs | filter $d.component == 'worker' AND $m.severity >= 5 | top 10 $d.error_message\n\n" +
			"### Escalation\n" +
			"- P1: Page on-call (critical job failures)\n" +
			"- P2: Create ticket (degraded performance)",
		Labels: map[string]string{
			"methodology": "RED",
			"tier":        "background",
		},
	},

	ComponentKubernetes: {
		ComponentType: ComponentKubernetes,
		Methodology:   MethodologyUSE,
		RecommendedMetrics: []MetricRecommendation{
			{
				Name:             "pod_restarts",
				Type:             "counter",
				Signal:           "errors",
				Query:            `source logs | filter $d.kubernetes exists AND $d.event_type == 'container_restart' | stats count() by $d.kubernetes.pod_name, bin(5m)`,
				DefaultThreshold: 3,
				Unit:             "restarts/5min",
				Description:      "Pod restarts indicate application crashes or OOM kills",
				BestPractices: []string{
					"Alert on restart rate, not just count",
					"Track OOMKilled vs CrashLoopBackOff separately",
				},
			},
			{
				Name:             "cpu_throttling",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.kubernetes exists | stats avg($d.cpu_throttled_percentage) as throttled by $d.kubernetes.pod_name, bin(1m)`,
				DefaultThreshold: 25,
				Unit:             "percentage",
				Description:      "CPU throttling indicates resource constraints",
				BestPractices: []string{
					"High throttling affects latency predictability",
					"Consider increasing CPU limits or optimizing application",
				},
			},
			{
				Name:             "memory_utilization",
				Type:             "gauge",
				Signal:           "utilization",
				Query:            `source logs | filter $d.kubernetes exists | stats avg($d.memory_usage_bytes / $d.memory_limit_bytes * 100) as utilization by $d.kubernetes.pod_name, bin(1m)`,
				DefaultThreshold: 90,
				Unit:             "percentage",
				Description:      "Memory utilization near limits risks OOMKill",
				BestPractices: []string{
					"Alert before OOMKill occurs (e.g., 85% warning)",
					"Track memory growth rate for leak detection",
				},
			},
			{
				Name:             "pending_pods",
				Type:             "gauge",
				Signal:           "saturation",
				Query:            `source logs | filter $d.kubernetes.pod_phase == 'Pending' | stats distinctcount($d.kubernetes.pod_name) as pending by bin(1m)`,
				DefaultThreshold: 5,
				Unit:             "pods",
				Description:      "Pending pods indicate scheduling constraints",
				BestPractices: []string{
					"Track pending duration, not just count",
					"Alert on pods pending > 5 minutes",
				},
			},
		},
		RunbookTemplate: "## Kubernetes Alert Runbook\n\n" +
			"### Initial Triage\n" +
			"1. Check cluster node health: kubectl get nodes\n" +
			"2. Review pod status: kubectl get pods -A | grep -v Running\n" +
			"3. Check resource quotas: kubectl describe resourcequota\n\n" +
			"### Investigation Steps\n" +
			"1. Check pod events: kubectl describe pod <pod-name>\n" +
			"2. Review container logs: kubectl logs <pod-name> --previous\n" +
			"3. Check resource usage: kubectl top pods\n\n" +
			"### Escalation\n" +
			"- P1: Page platform on-call (cluster-wide issues)\n" +
			"- P2: Create ticket for application team (single service)",
		Labels: map[string]string{
			"methodology": "USE",
			"tier":        "platform",
		},
	},
}

// CalculateBurnRate computes burn rate thresholds from an SLO target
// Reference: Google SRE Workbook Chapter 5 - Alerting on SLOs
func CalculateBurnRate(sloTarget float64, windowDays int) *BurnRateConfig {
	// Error budget = 1 - SLO target
	errorBudget := 1 - sloTarget

	// Monthly error budget in hours (assuming 30-day month)
	windowHours := float64(windowDays) * 24
	errorBudgetHours := errorBudget * windowHours

	config := &BurnRateConfig{
		SLO: SLOConfig{
			Target:             sloTarget,
			Window:             time.Duration(windowDays) * 24 * time.Hour,
			ErrorBudget:        errorBudget,
			MonthlyErrorBudget: errorBudgetHours,
		},
	}

	// Multi-window, Multi-Burn-Rate Alerting Strategy
	// Based on Google SRE recommendations
	//
	// Fast burn (Page): Consumes X% of error budget in Y time
	// Slow burn (Ticket): Consumes X% of error budget in Y time

	// Fast burn windows - for immediate attention (paging)
	config.FastBurnWindows = []BurnRateWindow{
		{
			// 2% budget consumption in 1 hour = 14.4x burn rate for 30-day window
			Duration:  1 * time.Hour,
			BurnRate:  14.4,
			Severity:  SeverityP1Critical,
			AlertType: "fast_burn",
		},
		{
			// 5% budget consumption in 6 hours = 6x burn rate
			Duration:  6 * time.Hour,
			BurnRate:  6.0,
			Severity:  SeverityP1Critical,
			AlertType: "fast_burn",
		},
	}

	// Slow burn windows - for ticketing
	config.SlowBurnWindows = []BurnRateWindow{
		{
			// 10% budget consumption in 24 hours = 3x burn rate
			Duration:  24 * time.Hour,
			BurnRate:  3.0,
			Severity:  SeverityP2Warning,
			AlertType: "slow_burn",
		},
		{
			// 10% budget consumption in 72 hours = 1x burn rate
			Duration:  72 * time.Hour,
			BurnRate:  1.0,
			Severity:  SeverityP3Info,
			AlertType: "slow_burn",
		},
	}

	return config
}

// CalculateErrorThreshold computes the error threshold for burn rate alerting
// Returns the error rate threshold that would consume the specified percentage
// of error budget in the given time window
func CalculateErrorThreshold(sloTarget float64, budgetConsumptionPercent float64, windowDuration time.Duration, sloWindowDays int) float64 {
	errorBudget := 1 - sloTarget
	sloWindowHours := float64(sloWindowDays) * 24
	alertWindowHours := windowDuration.Hours()

	// Error rate = (budget_consumption_% / 100) * error_budget * (slo_window / alert_window)
	threshold := (budgetConsumptionPercent / 100) * errorBudget * (sloWindowHours / alertWindowHours)

	return threshold
}

// GetBurnRateThreshold returns the error rate threshold for a given burn rate
func GetBurnRateThreshold(sloTarget float64, burnRate float64) float64 {
	errorBudget := 1 - sloTarget
	// Threshold = error_budget * burn_rate
	return errorBudget * burnRate
}

// FormatBurnRateExplanation generates a human-readable explanation of burn rate alerting
func FormatBurnRateExplanation(sloTarget float64, burnRate float64, windowDuration time.Duration, sloWindowDays int) string {
	errorBudget := 1 - sloTarget
	errorBudgetPercent := errorBudget * 100

	threshold := GetBurnRateThreshold(sloTarget, burnRate)
	thresholdPercent := threshold * 100

	// Calculate how much budget would be consumed
	sloWindowHours := float64(sloWindowDays) * 24
	alertWindowHours := windowDuration.Hours()
	budgetConsumed := (alertWindowHours / sloWindowHours) * burnRate * 100

	return fmt.Sprintf(
		"SLO: %.3f%% (Error Budget: %.4f%%)\n"+
			"Burn Rate: %.1fx\n"+
			"Alert Window: %s\n"+
			"Error Rate Threshold: %.4f%%\n"+
			"At this burn rate, %.1f%% of the %d-day error budget would be consumed in %s",
		sloTarget*100,
		errorBudgetPercent,
		burnRate,
		formatDuration(windowDuration),
		thresholdPercent,
		budgetConsumed,
		sloWindowDays,
		formatDuration(windowDuration),
	)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		days := d.Hours() / 24
		if days == float64(int(days)) {
			return fmt.Sprintf("%dd", int(days))
		}
		return fmt.Sprintf("%.1fd", days)
	}
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d >= time.Minute {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// ClassifySeverity determines alert severity based on impact and burn rate
func ClassifySeverity(isUserFacing bool, burnRate float64, componentType ComponentType) SeverityLevel {
	// P1 (Critical): User-facing + high burn rate (wake someone up)
	if isUserFacing && burnRate >= 6.0 {
		return SeverityP1Critical
	}

	// P1 for critical infrastructure with high burn rate
	criticalComponents := map[ComponentType]bool{
		ComponentDatabase:     true,
		ComponentMessageQueue: true,
		ComponentAPIGateway:   true,
	}
	if criticalComponents[componentType] && burnRate >= 10.0 {
		return SeverityP1Critical
	}

	// P2 (Warning): Saturation impending or medium burn rate
	if burnRate >= 1.0 || isUserFacing {
		return SeverityP2Warning
	}

	// P3 (Info): Low priority
	return SeverityP3Info
}

// GetMethodologyForComponent returns the recommended alerting methodology
func GetMethodologyForComponent(componentType ComponentType) AlertingMethodology {
	if strategy, ok := AlertingStrategyMatrix[componentType]; ok {
		return strategy.Methodology
	}
	// Default to RED for unknown components
	return MethodologyRED
}

// GetStrategyForComponent retrieves the full alerting strategy
func GetStrategyForComponent(componentType ComponentType) *AlertStrategyConfig {
	if strategy, ok := AlertingStrategyMatrix[componentType]; ok {
		return &strategy
	}
	return nil
}

// DetectComponentType attempts to infer component type from log patterns
func DetectComponentType(query string, useCase string) ComponentType {
	combined := strings.ToLower(query + " " + useCase)

	// Pattern matching for component detection - ordered by specificity
	// More specific patterns should come first to avoid false matches
	type patternGroup struct {
		compType ComponentType
		keywords []string
	}

	// Order matters! More specific patterns first
	patterns := []patternGroup{
		// Cache-specific patterns (check before database since both may have "redis")
		{ComponentCache, []string{"cache", "memcached", "hit rate", "miss rate", "eviction", "cache hit", "cache miss"}},
		// Kubernetes-specific patterns
		{ComponentKubernetes, []string{"kubernetes", "k8s", "pod", "kubectl", "deployment", "container restart"}},
		// Message queue patterns
		{ComponentMessageQueue, []string{"kafka", "rabbitmq", "sqs", "queue depth", "consumer lag", "consumer", "producer", "message queue", "dead letter"}},
		// API Gateway patterns
		{ComponentAPIGateway, []string{"gateway", "proxy", "routing", "upstream", "downstream", "rate limit"}},
		// Worker/Job patterns
		{ComponentWorker, []string{"worker", "job", "background job", "async", "cron", "scheduler", "task queue"}},
		// Serverless patterns
		{ComponentServerless, []string{"lambda", "serverless", "faas", "function invocation"}},
		// Database patterns (check after cache to avoid redis confusion)
		{ComponentDatabase, []string{"database", "db", "sql", "postgres", "mysql", "mongodb", "query duration", "slow query", "connection pool"}},
		// Web service patterns (general, check later)
		{ComponentWebService, []string{"http", "request", "response", "api", "endpoint", "rest", "graphql", "status_code"}},
		// Storage patterns
		{ComponentStorage, []string{"storage", "disk", "volume", "s3", "blob", "file system"}},
		// Network patterns
		{ComponentNetwork, []string{"network", "dns", "tcp", "socket", "connection timeout"}},
	}

	for _, pg := range patterns {
		for _, keyword := range pg.keywords {
			if strings.Contains(combined, keyword) {
				return pg.compType
			}
		}
	}

	return ComponentCustom
}

// ValidateActionability ensures an alert has required actionability fields
func ValidateActionability(suggestion *AdvancedAlertSuggestion) []string {
	var errors []string

	if suggestion.RunbookURL == "" {
		errors = append(errors, "Missing required field: runbook_url - Every alert must have a runbook")
	}

	if len(suggestion.SuggestedActions) == 0 {
		errors = append(errors, "Missing required field: suggested_actions - Alert must include actionable steps")
	}

	// Validate runbook URL format if provided
	if suggestion.RunbookURL != "" && !strings.HasPrefix(suggestion.RunbookURL, "http") && !strings.HasPrefix(suggestion.RunbookURL, "/") {
		errors = append(errors, "Invalid runbook_url format: must be a valid URL or path")
	}

	return errors
}

// GenerateRunbookURL creates a standardized runbook URL path
func GenerateRunbookURL(componentType ComponentType, alertName string) string {
	// Normalize alert name for URL
	normalized := strings.ToLower(alertName)
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")

	return fmt.Sprintf("/runbooks/%s/%s", componentType, normalized)
}

// GenerateDefaultActions creates default suggested actions for a component type
func GenerateDefaultActions(componentType ComponentType, signal string) []string {
	baseActions := []string{
		"1. Acknowledge alert and check current status",
		"2. Review recent changes (deployments, config changes)",
		"3. Check dependent services health",
	}

	// Add signal-specific actions
	switch signal {
	case "errors":
		baseActions = append(baseActions,
			"4. Query recent error logs to identify error patterns",
			"5. Check error rate trend to determine if improving or worsening",
			"6. Identify affected endpoints/users",
		)
	case "duration", "latency":
		baseActions = append(baseActions,
			"4. Check P50/P90/P99 latency distribution",
			"5. Identify slow endpoints or queries",
			"6. Check resource utilization (CPU, memory, I/O)",
		)
	case "saturation":
		baseActions = append(baseActions,
			"4. Check resource capacity and utilization",
			"5. Identify resource consumers",
			"6. Consider scaling or capacity increase",
		)
	case "utilization":
		baseActions = append(baseActions,
			"4. Verify utilization trend direction",
			"5. Identify top resource consumers",
			"6. Plan capacity increase if trend continues",
		)
	}

	// Add component-specific actions
	switch componentType {
	case ComponentDatabase:
		baseActions = append(baseActions,
			"7. Check active queries and locks",
			"8. Review slow query log",
			"9. Verify replication status if applicable",
		)
	case ComponentMessageQueue:
		baseActions = append(baseActions,
			"7. Check consumer lag and consumer health",
			"8. Review dead letter queue",
			"9. Verify broker cluster health",
		)
	case ComponentKubernetes:
		baseActions = append(baseActions,
			"7. kubectl get pods -n <namespace>",
			"8. kubectl describe pod <pod-name>",
			"9. kubectl logs <pod-name> --previous",
		)
	}

	return baseActions
}

// DynamicBaselineCalculator computes dynamic thresholds based on historical data
type DynamicBaselineCalculator struct {
	SeasonalityType  string
	StdDevMultiplier float64
	MinDataPoints    int
}

// NewDynamicBaselineCalculator creates a new baseline calculator
func NewDynamicBaselineCalculator(seasonalityType string, stdDevMultiplier float64) *DynamicBaselineCalculator {
	return &DynamicBaselineCalculator{
		SeasonalityType:  seasonalityType,
		StdDevMultiplier: stdDevMultiplier,
		MinDataPoints:    30, // Minimum data points for statistical significance
	}
}

// CalculateThreshold computes a dynamic threshold from historical data
func (d *DynamicBaselineCalculator) CalculateThreshold(historicalValues []float64) (lower, upper float64, err error) {
	if len(historicalValues) < d.MinDataPoints {
		return 0, 0, fmt.Errorf("insufficient data points: need %d, got %d", d.MinDataPoints, len(historicalValues))
	}

	// Calculate mean
	sum := 0.0
	for _, v := range historicalValues {
		sum += v
	}
	mean := sum / float64(len(historicalValues))

	// Calculate standard deviation
	sumSquaredDiff := 0.0
	for _, v := range historicalValues {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(historicalValues)))

	// Calculate bounds
	lower = mean - (d.StdDevMultiplier * stdDev)
	upper = mean + (d.StdDevMultiplier * stdDev)

	// Ensure lower bound is not negative for metrics that can't be negative
	if lower < 0 {
		lower = 0
	}

	return lower, upper, nil
}

// GenerateDynamicBaselineQuery creates a DataPrime query for baseline calculation
func GenerateDynamicBaselineQuery(metricField string, seasonalityType string, _ int) string {
	var groupBy string
	switch seasonalityType {
	case "hourly":
		groupBy = "hour_of_day"
	case "daily":
		groupBy = "day_of_week"
	case "weekly":
		groupBy = "week_of_year"
	default:
		groupBy = "hour_of_day"
	}

	return fmt.Sprintf(`source logs
| filter $d.%s exists
| extend %s = formatTimestamp($m.timestamp, '%s')
| stats
    avg($d.%s) as baseline_mean,
    stddev($d.%s) as baseline_stddev,
    count() as sample_count
  by %s
| filter sample_count >= 30`,
		metricField,
		groupBy,
		getTimestampFormat(seasonalityType),
		metricField,
		metricField,
		groupBy,
	)
}

// getTimestampFormat returns the appropriate timestamp format for seasonality
func getTimestampFormat(seasonalityType string) string {
	switch seasonalityType {
	case "hourly":
		return "HH"
	case "daily":
		return "E" // Day of week
	case "weekly":
		return "w" // Week of year
	default:
		return "HH"
	}
}
