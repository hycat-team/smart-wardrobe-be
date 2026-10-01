package grant

import (
	"sync"
	"time"

	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// GrantMetrics records every rejected event by reason code (FR-087).
//
// It exists as a separate type because a rejected event is the single most
// important thing to notice about this pipeline and the easiest to miss: the
// message is acknowledged, the consumer looks healthy, the queue depth is zero,
// and the entitlement simply never arrives. Without a counter keyed by reason,
// "an event was refused" and "an event was never sent" are indistinguishable —
// and a refund queue is the worst possible place to discover the difference.
type GrantMetrics struct {
	logger logger.Interface

	mu       sync.Mutex
	rejected map[string]int64
}

func NewGrantMetrics(log logger.Interface) *GrantMetrics {
	return &GrantMetrics{logger: log, rejected: make(map[string]int64)}
}

// RecordRejection advances the counter for one reason code.
//
// userID is carried on EVERY line, including for a payload that failed to parse.
// That is the point of this change (024 T114, SC-025): the observability table
// in quickstart.md GĐ8 has to be answerable from one log line — which account,
// which reason, at what moment — and a line carrying only the reason forces an
// operator to correlate against something else at exactly the moment nobody is
// watching. A payload too malformed to parse may have no usable id; the field is
// then empty rather than dropped, so the line shape stays stable to query.
//
// The warning is emitted on every rejection rather than throttled: unlike the
// degradation path, a rejected event is a data-integrity signal, not a load
// signal, and the volume is bounded by how often plans actually change. See the
// block above for what "alerting" does and does not mean in this service.
//
// 024 T175: the warning carries `rejected_total_for_reason` on every line, including
// the first. A first-occurrence line with a total of 1 is the one an operator wants to
// see — it is the earliest evidence — and it costs nothing extra to include.
// What "raise an alert when the counter rises" (FR-087) actually means here, stated
// plainly because the previous version of this comment implied a mechanism that does
// not exist.
//
// There is NO metrics backend. This package has no Prometheus client, the repository
// has none in any go.mod, and nothing scrapes these counters. What exists is:
//
//   - a process-local counter per reason code, reset when the process restarts;
//   - the same counters exposed on the AI service's health response under
//     `grant_rejected_by_reason` (bootstrap/router.go), which is what makes them
//     outlive a single log line rather than being RAM nobody can read; and
//   - one log line per rejection, which is what an operator's existing log alerting
//     consumes.
//
// So the alerting that exists is LOG-based and unconditional, not threshold-based.
// That is a deliberate choice, not an oversight: a rejected event is a data-integrity
// signal rather than a load signal, the volume is bounded by how often plans actually
// change, and the one genuinely high-volume reason — a duplicate from at-least-once
// delivery — is logged at Info rather than Warn precisely so it cannot drown the rest.
// Adding a numeric threshold would mean inventing an alerting policy that nobody has
// agreed to, and a threshold nobody tuned is worse than none: it either never fires or
// fires constantly, and an operator learns to ignore it.
//
// The gap against FR-087 is therefore real and worth naming: a restart loses the
// counts, and nothing fires without someone having set up log alerting. Closing that
// gap means adding a metrics backend, which is infrastructure rather than a code
// change, and is out of scope here.
//
// The threshold that DOES exist in this type is the duplicate exception just below —
// and that one is enforced, not aspirational.
func (m *GrantMetrics) RecordRejection(reason string, userID string) {
	m.mu.Lock()
	m.rejected[reason]++
	total := m.rejected[reason]
	m.mu.Unlock()

	fields := []zap.Field{
		zap.String("reason", reason),
		// Named user_id, matching every other log line in this service, so one
		// query joins them all.
		zap.String("user_id", userID),
		zap.Int64("rejected_total_for_reason", total),
		zap.Time("at", time.Now().UTC()),
	}
	// A duplicate is the expected consequence of at-least-once delivery, so it
	// is logged one level below a genuine refusal.
	if reason == ReasonDuplicateSourceEvent {
		m.logger.Info("subscription.plan.changed event was a duplicate", fields...)
		return
	}
	m.logger.Warn("subscription.plan.changed event rejected", fields...)
}

// Snapshot returns a copy of the counters.
func (m *GrantMetrics) Snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.rejected))
	for reason, count := range m.rejected {
		out[reason] = count
	}
	return out
}
