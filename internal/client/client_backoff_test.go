package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestClampedBackoff_NoOverflow verifies the exponential backoff calculation
// clamps to RetryWaitMax instead of overflowing time.Duration (int64
// nanoseconds) when RetryWaitMin is large and shift is large.
func TestClampedBackoff_NoOverflow(t *testing.T) {
	tests := []struct {
		name    string
		minWait time.Duration
		maxWait time.Duration
		shift   int
		want    time.Duration
	}{
		{
			name:    "small values - no clamp needed",
			minWait: 100 * time.Millisecond,
			maxWait: 30 * time.Second,
			shift:   2, // 100ms * 4 = 400ms
			want:    400 * time.Millisecond,
		},
		{
			name:    "shift zero - returns minWait",
			minWait: 1 * time.Second,
			maxWait: 30 * time.Second,
			shift:   0,
			want:    1 * time.Second,
		},
		{
			name:    "large minWait with large shift would overflow - clamps to max",
			minWait: time.Hour, // 3.6e12 ns
			maxWait: 30 * time.Second,
			shift:   30, // time.Hour << 30 overflows int64
			want:    30 * time.Second,
		},
		{
			name:    "moderate minWait with shift 30 exceeds max - clamps",
			minWait: 1 * time.Second,
			maxWait: 30 * time.Second,
			shift:   30, // 1s * 2^30 is huge, far exceeds 30s
			want:    30 * time.Second,
		},
		{
			name:    "minWait already exceeds maxWait at shift 0",
			minWait: 1 * time.Minute,
			maxWait: 30 * time.Second,
			shift:   0,
			want:    30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampedBackoff(tt.minWait, tt.maxWait, tt.shift)
			assert.Equal(t, tt.want, got)
			assert.GreaterOrEqual(t, got, time.Duration(0), "backoff must never be negative (overflow symptom)")
		})
	}
}
