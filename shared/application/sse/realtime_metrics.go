package sse

import "sync/atomic"

// In-memory realtime counters/gauges (feature 023, FR-026). They mirror the
// messaging metrics pattern (atomic + a Get* snapshot) so operations can detect
// silently dropped events under the best-effort policy (FR-025).
var (
	realtimePublishTotal  atomic.Int64
	realtimePublishFailed atomic.Int64
	realtimeRelayTotal    atomic.Int64
	realtimeRelayFailed   atomic.Int64
	realtimeSubscribers   atomic.Int64
)

// IncRealtimePublishTotal counts every publish attempt.
func IncRealtimePublishTotal() { realtimePublishTotal.Add(1) }

// IncRealtimePublishFailed counts failed publish attempts (Redis down/timeout).
func IncRealtimePublishFailed() { realtimePublishFailed.Add(1) }

// IncRealtimeRelayTotal counts events relayed into the local SSEManager.
func IncRealtimeRelayTotal() { realtimeRelayTotal.Add(1) }

// IncRealtimeRelayFailed counts relay-side failures.
func IncRealtimeRelayFailed() { realtimeRelayFailed.Add(1) }

// IncRealtimeSubscribers tracks an opened SSE subscription.
func IncRealtimeSubscribers() { realtimeSubscribers.Add(1) }

// DecRealtimeSubscribers tracks a closed SSE subscription.
func DecRealtimeSubscribers() { realtimeSubscribers.Add(-1) }

// GetRealtimeMetrics returns a point-in-time snapshot for /health or /metrics.
func GetRealtimeMetrics() map[string]int64 {
	return map[string]int64{
		"realtime_publish_total":  realtimePublishTotal.Load(),
		"realtime_publish_failed": realtimePublishFailed.Load(),
		"realtime_relay_total":    realtimeRelayTotal.Load(),
		"realtime_relay_failed":   realtimeRelayFailed.Load(),
		"realtime_subscribers":    realtimeSubscribers.Load(),
	}
}
