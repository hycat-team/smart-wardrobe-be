package grant

import (
	"context"
	"sync"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/shared/domain/constants/aifallbackreason"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// tier-2 warnings are throttled on powers of two rather than emitted per
// request: a warning on every degraded call would flood the log during an
// incident, which is exactly when the log is most needed for other things. The
// counter still advances every time; only the warning is throttled, so the log
// still shows the rate doubling.
const tier2WarningStep = 1

// FallbackMetrics is the observer that records every degradation.
//
// It exists as its own type rather than inline logging because FR-010 requires
// the counts to be retrievable: "the system fell back" and "the model provider
// was down" look identical in a log line, and an operator cannot tell which knob
// to turn without a number that separates them.
type FallbackMetrics struct {
	logger logger.Interface

	mu       sync.Mutex
	counters map[aifallbackreason.Reason]int64
	tier2    int64
}

func NewFallbackMetrics(log logger.Interface) *FallbackMetrics {
	return &FallbackMetrics{
		logger:   log,
		counters: make(map[aifallbackreason.Reason]int64),
	}
}

// OnFallback records one degradation and decides whether to warn.
func (m *FallbackMetrics) OnFallback(_ context.Context, userID uuid.UUID, operation aioperation.AIOperation, reason aifallbackreason.Reason, tier2 bool) {
	now := time.Now().UTC()

	m.mu.Lock()
	m.counters[reason]++
	total := m.counters[reason]
	if tier2 {
		m.tier2++
	}
	tier2Total := m.tier2
	m.mu.Unlock()

	fields := []zap.Field{
		zap.String("user_id", userID.String()),
		zap.String("operation", string(operation)),
		zap.String("fallback_reason", string(reason)),
		zap.Bool("tier2", tier2),
		zap.Int64("fallback_count_for_reason", total),
		zap.Time("at", now),
	}

	switch {
	case tier2 && isWarningStep(tier2Total):
		// Tier 2 means a paid feature is running on the free model. That is a
		// revenue and quality problem, not a routine event, so it is escalated
		// above the per-request info line.
		m.logger.Warn("AI cost policy fallback tier 2 in use, running on the free model", fields...)
	case tier2:
		m.logger.Info("AI cost policy fallback tier 2 in use", fields...)
	default:
		m.logger.Warn("AI cost policy fallback tier 1 in use, account has no policy grant", fields...)
	}
}

// isWarningStep reports whether a running total has just reached a warning
// boundary: the first occurrence, then every doubling.
func isWarningStep(total int64) bool {
	if total <= int64(tier2WarningStep) {
		return true
	}
	return total&(total-1) == 0
}

// Snapshot returns a copy of the current counters, for /metrics and tests.
func (m *FallbackMetrics) Snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.counters)+1)
	for reason, count := range m.counters {
		out[string(reason)] = count
	}
	out["tier2_total"] = m.tier2
	return out
}
