// Package metrics provides metrics collection and reporting for the MCP server.
package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// Prometheus metric labels
const (
	labelTool   = "tool"
	labelStatus = "status"
	labelType   = "type"
)

// Metrics tracks operational metrics with both internal counters and Prometheus metrics
type Metrics struct {
	// Request metrics (internal atomic counters for fast access)
	totalRequests      atomic.Uint64
	successfulRequests atomic.Uint64
	failedRequests     atomic.Uint64
	retriedRequests    atomic.Uint64

	// Latency tracking
	totalLatency atomic.Int64 // microseconds
	latencyCount atomic.Uint64
	maxLatency   atomic.Int64
	minLatency   atomic.Int64

	// Rate limiting metrics
	rateLimitHits atomic.Uint64

	// Error tracking by status code
	errorsMu       sync.RWMutex
	errorsByStatus map[int]uint64

	// Tool usage tracking
	toolsMu     sync.RWMutex
	toolUsage   map[string]uint64
	toolErrors  map[string]uint64
	toolLatency map[string]int64 // microseconds

	logger *zap.Logger

	// Prometheus metrics
	promRequestsTotal      prometheus.Counter
	promRequestsSuccessful prometheus.Counter
	promRequestsFailed     prometheus.Counter
	promRequestsRetried    prometheus.Counter
	promRateLimitHits      prometheus.Counter
	promRequestLatency     prometheus.Histogram
	promErrorsByStatus     *prometheus.CounterVec
	promToolCalls          *prometheus.CounterVec
	promToolErrors         *prometheus.CounterVec
	promToolLatency        *prometheus.HistogramVec
}

// New creates a new metrics tracker with Prometheus integration
func New(logger *zap.Logger) *Metrics {
	m := &Metrics{
		errorsByStatus: make(map[int]uint64),
		toolUsage:      make(map[string]uint64),
		toolErrors:     make(map[string]uint64),
		toolLatency:    make(map[string]int64),
		logger:         logger,

		// Initialize Prometheus metrics using promauto (auto-registers with default registry)
		promRequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "requests_total",
			Help:      "Total number of API requests made to IBM Cloud Logs",
		}),
		promRequestsSuccessful: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "requests_successful_total",
			Help:      "Total number of successful API requests",
		}),
		promRequestsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "requests_failed_total",
			Help:      "Total number of failed API requests",
		}),
		promRequestsRetried: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "requests_retried_total",
			Help:      "Total number of retried API requests",
		}),
		promRateLimitHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "rate_limit_hits_total",
			Help:      "Total number of rate limit hits",
		}),
		promRequestLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "logs_mcp",
			Name:      "request_latency_seconds",
			Help:      "API request latency in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		}),
		promErrorsByStatus: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "errors_by_status_total",
			Help:      "Errors by HTTP status code",
		}, []string{labelStatus}),

		// Tool-specific metrics - tracks every tool call with labels for tool name
		promToolCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "tool_calls_total",
			Help:      "Total number of tool calls, labeled by tool name (e.g., query_logs, create_alert, list_dashboards)",
		}, []string{labelTool}),
		promToolErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "logs_mcp",
			Name:      "tool_errors_total",
			Help:      "Total number of tool errors, labeled by tool name",
		}, []string{labelTool}),
		promToolLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "logs_mcp",
			Name:      "tool_latency_seconds",
			Help:      "Tool execution latency in seconds, labeled by tool name",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		}, []string{labelTool}),
	}

	// Initialize min latency to max value
	m.minLatency.Store(int64(time.Hour))

	return m
}

// RecordRequest records a request (both internal counters and Prometheus)
func (m *Metrics) RecordRequest(success bool, latency time.Duration, statusCode int) {
	// Update internal counters
	m.totalRequests.Add(1)

	// Update Prometheus counters
	m.promRequestsTotal.Inc()
	m.promRequestLatency.Observe(latency.Seconds())

	if success {
		m.successfulRequests.Add(1)
		m.promRequestsSuccessful.Inc()
	} else {
		m.failedRequests.Add(1)
		m.promRequestsFailed.Inc()
		m.recordErrorStatus(statusCode)
	}

	m.recordLatency(latency)
}

// RecordRetry records a retry attempt
func (m *Metrics) RecordRetry() {
	m.retriedRequests.Add(1)
	m.promRequestsRetried.Inc()
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit() {
	m.rateLimitHits.Add(1)
	m.promRateLimitHits.Inc()
}

// RecordToolExecution records tool usage (both internal counters and Prometheus)
// This is called for every tool invocation, tracking:
// - Total calls per tool
// - Errors per tool
// - Latency distribution per tool
func (m *Metrics) RecordToolExecution(toolName string, success bool, latency time.Duration) {
	// Update internal counters
	m.toolsMu.Lock()
	m.toolUsage[toolName]++
	if !success {
		m.toolErrors[toolName]++
	}

	// Update average latency using rolling average to avoid integer overflow
	if latency > 0 && m.toolUsage[toolName] > 0 {
		currentLatency := m.toolLatency[toolName]
		// Use float64 for calculation to avoid integer overflow issues
		count := float64(m.toolUsage[toolName])
		avgLatency := (float64(currentLatency)*(count-1) + float64(latency.Microseconds())) / count
		m.toolLatency[toolName] = int64(avgLatency)
	}
	m.toolsMu.Unlock()

	// Update Prometheus metrics (labeled by tool name)
	m.promToolCalls.WithLabelValues(toolName).Inc()
	m.promToolLatency.WithLabelValues(toolName).Observe(latency.Seconds())
	if !success {
		m.promToolErrors.WithLabelValues(toolName).Inc()
	}
}

func (m *Metrics) recordLatency(latency time.Duration) {
	latencyUs := latency.Microseconds()

	m.totalLatency.Add(latencyUs)
	m.latencyCount.Add(1)

	// Update max latency
	for {
		currentMax := m.maxLatency.Load()
		if latencyUs <= currentMax {
			break
		}
		if m.maxLatency.CompareAndSwap(currentMax, latencyUs) {
			break
		}
	}

	// Update min latency
	for {
		currentMin := m.minLatency.Load()
		if latencyUs >= currentMin {
			break
		}
		if m.minLatency.CompareAndSwap(currentMin, latencyUs) {
			break
		}
	}
}

func (m *Metrics) recordErrorStatus(statusCode int) {
	if statusCode == 0 {
		return
	}

	m.errorsMu.Lock()
	m.errorsByStatus[statusCode]++
	m.errorsMu.Unlock()

	// Update Prometheus counter with status code label
	m.promErrorsByStatus.WithLabelValues(fmt.Sprintf("%d", statusCode)).Inc()
}

// GetStats returns current statistics
func (m *Metrics) GetStats() Stats {
	m.errorsMu.RLock()
	errorsByStatus := make(map[int]uint64, len(m.errorsByStatus))
	for k, v := range m.errorsByStatus {
		errorsByStatus[k] = v
	}
	m.errorsMu.RUnlock()

	m.toolsMu.RLock()
	toolUsage := make(map[string]uint64, len(m.toolUsage))
	toolErrors := make(map[string]uint64, len(m.toolErrors))
	toolLatency := make(map[string]time.Duration, len(m.toolLatency))
	for k, v := range m.toolUsage {
		toolUsage[k] = v
	}
	for k, v := range m.toolErrors {
		toolErrors[k] = v
	}
	for k, v := range m.toolLatency {
		toolLatency[k] = time.Duration(v) * time.Microsecond
	}
	m.toolsMu.RUnlock()

	totalReq := m.totalRequests.Load()
	latencyCount := m.latencyCount.Load()

	var avgLatency time.Duration
	if latencyCount > 0 {
		// Use float64 division to avoid integer overflow issues
		avgLatencyMicros := float64(m.totalLatency.Load()) / float64(latencyCount)
		avgLatency = time.Duration(avgLatencyMicros) * time.Microsecond
	}

	return Stats{
		TotalRequests:      totalReq,
		SuccessfulRequests: m.successfulRequests.Load(),
		FailedRequests:     m.failedRequests.Load(),
		RetriedRequests:    m.retriedRequests.Load(),
		RateLimitHits:      m.rateLimitHits.Load(),
		AverageLatency:     avgLatency,
		MaxLatency:         time.Duration(m.maxLatency.Load()) * time.Microsecond,
		MinLatency:         time.Duration(m.minLatency.Load()) * time.Microsecond,
		ErrorsByStatus:     errorsByStatus,
		ToolUsage:          toolUsage,
		ToolErrors:         toolErrors,
		ToolLatency:        toolLatency,
	}
}

// LogStats logs current statistics
func (m *Metrics) LogStats() {
	stats := m.GetStats()

	var errorRate float64
	if stats.TotalRequests > 0 {
		errorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
	}

	m.logger.Info("Operational metrics",
		zap.Uint64("total_requests", stats.TotalRequests),
		zap.Uint64("successful_requests", stats.SuccessfulRequests),
		zap.Uint64("failed_requests", stats.FailedRequests),
		zap.Float64("error_rate_pct", errorRate),
		zap.Uint64("retried_requests", stats.RetriedRequests),
		zap.Uint64("rate_limit_hits", stats.RateLimitHits),
		zap.Duration("avg_latency", stats.AverageLatency),
		zap.Duration("max_latency", stats.MaxLatency),
		zap.Duration("min_latency", stats.MinLatency),
		zap.Any("errors_by_status", stats.ErrorsByStatus),
		zap.Any("tool_usage", stats.ToolUsage),
	)
}

// Stats represents current metrics
type Stats struct {
	TotalRequests      uint64
	SuccessfulRequests uint64
	FailedRequests     uint64
	RetriedRequests    uint64
	RateLimitHits      uint64
	AverageLatency     time.Duration
	MaxLatency         time.Duration
	MinLatency         time.Duration
	ErrorsByStatus     map[int]uint64
	ToolUsage          map[string]uint64
	ToolErrors         map[string]uint64
	ToolLatency        map[string]time.Duration
}

// GetPrometheusRegistry returns the default Prometheus registry
// This can be used with promhttp.HandlerFor() to serve metrics
func GetPrometheusRegistry() *prometheus.Registry {
	// Return the default registry which promauto uses
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}
