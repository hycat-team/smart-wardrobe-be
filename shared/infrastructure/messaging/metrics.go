package messaging

import "sync/atomic"

var (
	publishTotal     atomic.Int64
	publishFailed    atomic.Int64
	consumeAckTotal  atomic.Int64
	consumeNackTotal atomic.Int64
	reconnectTotal   atomic.Int64

	// Gauges — cập nhật định kỳ bởi OutboxWorker.RefreshGauges.
	outboxPending atomic.Int64
	queueDepth    atomic.Int64
	dlqDepth      atomic.Int64
)

func incPublishTotal()  { publishTotal.Add(1) }
func incPublishFailed() { publishFailed.Add(1) }
func incAck()           { consumeAckTotal.Add(1) }
func incNack()          { consumeNackTotal.Add(1) }
func incReconnect()     { reconnectTotal.Add(1) }

// SetGauges cập nhật các gauge vận hành (được /metrics và health check đọc).
func SetGauges(outboxPendingCount, queueDepthCount, dlqDepthCount int64) {
	outboxPending.Store(outboxPendingCount)
	queueDepth.Store(queueDepthCount)
	dlqDepth.Store(dlqDepthCount)
}

// GetMetrics returns current counters and gauges for /metrics or structured log.
func GetMetrics() map[string]int64 {
	return map[string]int64{
		"publish_total":      publishTotal.Load(),
		"publish_failed":     publishFailed.Load(),
		"consume_ack_total":  consumeAckTotal.Load(),
		"consume_nack_total": consumeNackTotal.Load(),
		"reconnect_total":    reconnectTotal.Load(),
		"outbox_pending":     outboxPending.Load(),
		"queue_depth":        queueDepth.Load(),
		"dlq_depth":          dlqDepth.Load(),
	}
}
