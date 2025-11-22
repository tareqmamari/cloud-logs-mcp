package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Metrics tracks operational metrics
type Metrics struct {
	// Request metrics
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
}

// New creates a new metrics tracker
func New(logger *zap.Logger) *Metrics {
	m := &Metrics{
		errorsByStatus: make(map[int]uint64),
		toolUsage:      make(map[string]uint64),
		toolErrors:     make(map[string]uint64),
		toolLatency:    make(map[string]int64),
		logger:         logger,
	}

	// Initialize min latency to max value
	m.minLatency.Store(int64(time.Hour))

	return m
}

// RecordRequest records a request
func (m *Metrics) RecordRequest(success bool, latency time.Duration, statusCode int) {
	m.totalRequests.Add(1)

	if success {
		m.successfulRequests.Add(1)
	} else {
		m.failedRequests.Add(1)
		m.recordErrorStatus(statusCode)
	}

	m.recordLatency(latency)
}

// RecordRetry records a retry attempt
func (m *Metrics) RecordRetry() {
	m.retriedRequests.Add(1)
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit() {
	m.rateLimitHits.Add(1)
}

// RecordToolExecution records tool usage
func (m *Metrics) RecordToolExecution(toolName string, success bool, latency time.Duration) {
	m.toolsMu.Lock()
	defer m.toolsMu.Unlock()

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
	defer m.errorsMu.Unlock()
	m.errorsByStatus[statusCode]++
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
