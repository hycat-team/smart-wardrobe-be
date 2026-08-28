package messaging

import "sync/atomic"

var (
	publishTotal     atomic.Int64
	publishFailed    atomic.Int64
	consumeAckTotal  atomic.Int64
	consumeNackTotal atomic.Int64
	reconnectTotal   atomic.Int64
)

func incPublishTotal()  { publishTotal.Add(1) }
func incPublishFailed() { publishFailed.Add(1) }
func incAck()           { consumeAckTotal.Add(1) }
func incNack()          { consumeNackTotal.Add(1) }
func incReconnect()     { reconnectTotal.Add(1) }

// GetMetrics returns current counters for /metrics or structured log.
func GetMetrics() map[string]int64 {
	return map[string]int64{
		"publish_total":      publishTotal.Load(),
		"publish_failed":     publishFailed.Load(),
		"consume_ack_total":  consumeAckTotal.Load(),
		"consume_nack_total": consumeNackTotal.Load(),
		"reconnect_total":    reconnectTotal.Load(),
	}
}
