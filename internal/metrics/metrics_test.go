package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// newTestMetrics creates a Metrics instance with an isolated Prometheus registry.
func newTestMetrics(t *testing.T) *Metrics {
	t.Helper()
	reg := prometheus.NewRegistry()
	return NewWithRegistry(zap.NewNop(), reg)
}

func TestRecordRequest_Success(t *testing.T) {
	m := newTestMetrics(t)

	m.RecordRequest(true, 100*time.Millisecond, 200)

	stats := m.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("TotalRequests = %d, want 1", stats.TotalRequests)
	}
	if stats.SuccessfulRequests != 1 {
		t.Errorf("SuccessfulRequests = %d, want 1", stats.SuccessfulRequests)
	}
	if stats.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0", stats.FailedRequests)
	}
	if stats.AverageLatency != 100*time.Millisecond {
		t.Errorf("AverageLatency = %v, want 100ms", stats.AverageLatency)
	}
}

func TestRecordRequest_Failure(t *testing.T) {
	m := newTestMetrics(t)

	m.RecordRequest(false, 50*time.Millisecond, 500)
	m.RecordRequest(false, 60*time.Millisecond, 429)
	m.RecordRequest(false, 70*time.Millisecond, 500)

	stats := m.GetStats()
	if stats.TotalRequests != 3 {
		t.Errorf("TotalRequests = %d, want 3", stats.TotalRequests)
	}
	if stats.FailedRequests != 3 {
		t.Errorf("FailedRequests = %d, want 3", stats.FailedRequests)
	}
	if stats.SuccessfulRequests != 0 {
		t.Errorf("SuccessfulRequests = %d, want 0", stats.SuccessfulRequests)
	}
	if got, want := stats.ErrorsByStatus[500], uint64(2); got != want {
		t.Errorf("ErrorsByStatus[500] = %d, want %d", got, want)
	}
	if got, want := stats.ErrorsByStatus[429], uint64(1); got != want {
		t.Errorf("ErrorsByStatus[429] = %d, want %d", got, want)
	}
}

func TestRecordToolExecution(t *testing.T) {
	m := newTestMetrics(t)

	m.RecordToolExecution("query_logs", true, 200*time.Millisecond)
	m.RecordToolExecution("query_logs", false, 300*time.Millisecond)
	m.RecordToolExecution("create_alert", true, 150*time.Millisecond)

	stats := m.GetStats()

	if got, want := stats.ToolUsage["query_logs"], uint64(2); got != want {
		t.Errorf("ToolUsage[query_logs] = %d, want %d", got, want)
	}
	if got, want := stats.ToolUsage["create_alert"], uint64(1); got != want {
		t.Errorf("ToolUsage[create_alert] = %d, want %d", got, want)
	}
	if got, want := stats.ToolErrors["query_logs"], uint64(1); got != want {
		t.Errorf("ToolErrors[query_logs] = %d, want %d", got, want)
	}
	if stats.ToolErrors["create_alert"] != 0 {
		t.Errorf("ToolErrors[create_alert] = %d, want 0", stats.ToolErrors["create_alert"])
	}
	if stats.ToolLatency["query_logs"] == 0 {
		t.Error("ToolLatency[query_logs] should be non-zero")
	}
	if stats.ToolLatency["create_alert"] == 0 {
		t.Error("ToolLatency[create_alert] should be non-zero")
	}
}

func TestRecordToolExecution_LatencyRollingAverage(t *testing.T) {
	m := newTestMetrics(t)

	// Record three calls with known latencies: 100ms, 200ms, 300ms
	// Rolling average after each: 100ms, 150ms, 200ms
	m.RecordToolExecution("tool_a", true, 100*time.Millisecond)
	m.RecordToolExecution("tool_a", true, 200*time.Millisecond)
	m.RecordToolExecution("tool_a", true, 300*time.Millisecond)

	stats := m.GetStats()
	avgLatency := stats.ToolLatency["tool_a"]

	// The rolling average of 100ms, 200ms, 300ms should be 200ms.
	// Allow a small tolerance for floating point rounding in microsecond conversion.
	expected := 200 * time.Millisecond
	tolerance := 1 * time.Millisecond
	if avgLatency < expected-tolerance || avgLatency > expected+tolerance {
		t.Errorf("ToolLatency rolling average = %v, want ~%v", avgLatency, expected)
	}
}

func TestRecordRetry(t *testing.T) {
	m := newTestMetrics(t)

	m.RecordRetry()
	m.RecordRetry()
	m.RecordRetry()

	stats := m.GetStats()
	if stats.RetriedRequests != 3 {
		t.Errorf("RetriedRequests = %d, want 3", stats.RetriedRequests)
	}
}

func TestRecordRateLimitHit(t *testing.T) {
	m := newTestMetrics(t)

	m.RecordRateLimitHit()
	m.RecordRateLimitHit()

	stats := m.GetStats()
	if stats.RateLimitHits != 2 {
		t.Errorf("RateLimitHits = %d, want 2", stats.RateLimitHits)
	}
}

func TestGetStats_Empty(t *testing.T) {
	m := newTestMetrics(t)

	stats := m.GetStats()

	if stats.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", stats.TotalRequests)
	}
	if stats.SuccessfulRequests != 0 {
		t.Errorf("SuccessfulRequests = %d, want 0", stats.SuccessfulRequests)
	}
	if stats.FailedRequests != 0 {
		t.Errorf("FailedRequests = %d, want 0", stats.FailedRequests)
	}
	if stats.RetriedRequests != 0 {
		t.Errorf("RetriedRequests = %d, want 0", stats.RetriedRequests)
	}
	if stats.RateLimitHits != 0 {
		t.Errorf("RateLimitHits = %d, want 0", stats.RateLimitHits)
	}
	if stats.AverageLatency != 0 {
		t.Errorf("AverageLatency = %v, want 0", stats.AverageLatency)
	}
	if stats.MaxLatency != 0 {
		t.Errorf("MaxLatency = %v, want 0", stats.MaxLatency)
	}
	if len(stats.ErrorsByStatus) != 0 {
		t.Errorf("ErrorsByStatus should be empty, got %v", stats.ErrorsByStatus)
	}
	if len(stats.ToolUsage) != 0 {
		t.Errorf("ToolUsage should be empty, got %v", stats.ToolUsage)
	}
	if len(stats.ToolErrors) != 0 {
		t.Errorf("ToolErrors should be empty, got %v", stats.ToolErrors)
	}
}

func TestGetStats_LatencyMinMax(t *testing.T) {
	m := newTestMetrics(t)

	latencies := []time.Duration{
		50 * time.Millisecond,
		10 * time.Millisecond,
		200 * time.Millisecond,
		75 * time.Millisecond,
	}
	for _, lat := range latencies {
		m.RecordRequest(true, lat, 200)
	}

	stats := m.GetStats()

	if stats.MinLatency != 10*time.Millisecond {
		t.Errorf("MinLatency = %v, want 10ms", stats.MinLatency)
	}
	if stats.MaxLatency != 200*time.Millisecond {
		t.Errorf("MaxLatency = %v, want 200ms", stats.MaxLatency)
	}
}

func TestLogStats(t *testing.T) {
	m := newTestMetrics(t)

	// Record some data so LogStats exercises non-zero paths.
	m.RecordRequest(true, 100*time.Millisecond, 200)
	m.RecordRequest(false, 50*time.Millisecond, 503)

	// LogStats should not panic.
	m.LogStats()
}

func TestConcurrentAccess(t *testing.T) {
	m := newTestMetrics(t)

	const goroutines = 10
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m.RecordRequest(j%2 == 0, time.Duration(j)*time.Millisecond, 200+j%5)
				m.RecordRetry()
				m.RecordRateLimitHit()
				m.RecordToolExecution("concurrent_tool", j%3 == 0, time.Duration(j)*time.Millisecond)
				_ = m.GetStats()
			}
		}()
	}

	wg.Wait()

	stats := m.GetStats()
	expectedTotal := uint64(goroutines * iterations)
	if stats.TotalRequests != expectedTotal {
		t.Errorf("TotalRequests = %d, want %d", stats.TotalRequests, expectedTotal)
	}
	if stats.RetriedRequests != expectedTotal {
		t.Errorf("RetriedRequests = %d, want %d", stats.RetriedRequests, expectedTotal)
	}
	if stats.RateLimitHits != expectedTotal {
		t.Errorf("RateLimitHits = %d, want %d", stats.RateLimitHits, expectedTotal)
	}
	if stats.ToolUsage["concurrent_tool"] != expectedTotal {
		t.Errorf("ToolUsage[concurrent_tool] = %d, want %d", stats.ToolUsage["concurrent_tool"], expectedTotal)
	}
}

func TestGetPrometheusRegistry(t *testing.T) {
	reg := GetPrometheusRegistry()
	if reg == nil {
		t.Fatal("GetPrometheusRegistry() returned nil")
	}
	// Verify it returns a *prometheus.Registry (the default one).
	if _, ok := interface{}(reg).(*prometheus.Registry); !ok {
		t.Fatalf("GetPrometheusRegistry() returned %T, want *prometheus.Registry", reg)
	}
}
