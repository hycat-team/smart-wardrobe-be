package orchestrator

import (
	"sync"

	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// providerGemini names the provider whose vision call reports real usage. The
// OpenAI-compatible vision response carries no token counters, so a zero count
// there is expected and must not be reported as a metering fault. It mirrors
// shared/infrastructure/ai.ProviderGemini; the value is repeated here because
// the application layer must not import the transport package.
const providerGemini = "google"

// ceilingObservation is how one request compares to the token ceilings the
// policy resolved for it.
//
// The columns normal_max_input_tokens / reduced_max_input_tokens have been part
// of the policy since 024 but nothing ever compared them to the estimate, so a
// prompt could exceed its declared ceiling and still be sent. Phase 1 makes the
// gap visible instead of silently closing it: the ceilings are read from the
// grant snapshot, and the snapshots currently in production still carry numbers
// that a legitimate outfit prompt already exceeds, so enforcing now would
// degrade paying users onto the free route.
type ceilingObservation struct {
	Operation       string
	EstimatedInput  int64
	MaxInputTokens  int
	RequestedOutput int
	MaxOutputTokens int
}

func newCeilingObservation(plan *uc_interfaces.ExecutionPlan, estimatedInput int64, requestedOutput int) ceilingObservation {
	if plan == nil {
		return ceilingObservation{EstimatedInput: estimatedInput, RequestedOutput: requestedOutput}
	}
	return ceilingObservation{
		Operation:       string(plan.Operation),
		EstimatedInput:  estimatedInput,
		MaxInputTokens:  plan.MaxInputTokens,
		RequestedOutput: requestedOutput,
		MaxOutputTokens: plan.MaxOutputTokens,
	}
}

// inputExceeded reports whether the prompt is over the ceiling. A ceiling of
// zero means "no ceiling resolved" — operations that are always free (vision,
// embedding) return an early plan without one, and must never be flagged.
func (o ceilingObservation) inputExceeded() bool {
	return o.MaxInputTokens > 0 && o.EstimatedInput > int64(o.MaxInputTokens)
}

func (o ceilingObservation) outputExceeded() bool {
	return o.MaxOutputTokens > 0 && o.RequestedOutput > o.MaxOutputTokens
}

func (o ceilingObservation) exceeded() bool {
	return o.inputExceeded() || o.outputExceeded()
}

func (o ceilingObservation) inputRatio() float64 {
	if o.MaxInputTokens <= 0 {
		return 0
	}
	return float64(o.EstimatedInput) / float64(o.MaxInputTokens)
}

// ceilingMetrics counts ceiling breaches and throttles the warning on powers of
// two, following grant.FallbackMetrics: during an incident every outfit call can
// breach the ceiling, and one log line per call would bury everything else.
// The counter still advances on every breach, so the log shows the rate
// doubling rather than every event.
type ceilingMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
}

func newCeilingMetrics() *ceilingMetrics {
	return &ceilingMetrics{counters: make(map[string]int64)}
}

// tokenCeilings is package-scoped because the orchestrator is constructed by
// wire without an observability dependency. Tests build their own instance.
var tokenCeilings = newCeilingMetrics()

// onEvent counts one occurrence and reports whether this call should log.
func (m *ceilingMetrics) onEvent(log logger.Interface, key, message string, fields ...zap.Field) {
	m.mu.Lock()
	m.counters[key]++
	count := m.counters[key]
	m.mu.Unlock()

	if count&(count-1) != 0 {
		return
	}
	log.Warn(message, append(fields, zap.Int64("occurrences", count))...)
}

// onExceeded records one ceiling breach.
func (m *ceilingMetrics) onExceeded(log logger.Interface, observation ceilingObservation, userID uuid.UUID) {
	m.onEvent(log, "ceiling:"+observation.Operation, "AI prompt exceeds its resolved token ceiling",
		zap.String("operation", observation.Operation),
		zap.String("user_id", userID.String()),
		zap.Int64("estimated_input_tokens", observation.EstimatedInput),
		zap.Int("max_input_tokens", observation.MaxInputTokens),
		zap.Float64("input_ratio", observation.inputRatio()),
		zap.Int("requested_output_tokens", observation.RequestedOutput),
		zap.Int("max_output_tokens", observation.MaxOutputTokens),
	)
}

// onOutputOverrun records one response that produced more tokens than the
// policy allows for the operation.
func (m *ceilingMetrics) onOutputOverrun(log logger.Interface, observation ceilingObservation, userID uuid.UUID) {
	m.onEvent(log, "output:"+observation.Operation, "AI response exceeded the policy output ceiling",
		zap.String("operation", observation.Operation),
		zap.String("user_id", userID.String()),
		zap.Int("output_tokens", observation.RequestedOutput),
		zap.Int("max_output_tokens", observation.MaxOutputTokens),
	)
}

// onVisionUsageMissing records a Gemini vision call that reported no tokens.
func (m *ceilingMetrics) onVisionUsageMissing(log logger.Interface, operation string, userID uuid.UUID) {
	m.onEvent(log, "vision-usage:"+operation, "Gemini vision response carried no token usage; vision cost is not metered",
		zap.String("operation", operation),
		zap.String("user_id", userID.String()),
	)
}

// observeTokenCeilings logs a breach of the resolved ceilings. It is
// observation only: the request keeps its route, its options and its output
// budget, so this cannot change what a customer receives.
func (uc *OrchestratorUseCase) observeTokenCeilings(plan *uc_interfaces.ExecutionPlan, estimatedInput int64, requestedOutput int, userID uuid.UUID) {
	observation := newCeilingObservation(plan, estimatedInput, requestedOutput)
	if !observation.exceeded() {
		return
	}
	tokenCeilings.onExceeded(uc.logger, observation, userID)
}

// observeOutputUsage compares the tokens the provider actually produced with
// the ceiling the policy declared. It is the second half of phase 1: the
// requested budget and the delivered budget can disagree (the commerce config
// asks for 600 output tokens while the free policy allows 300), and the policy
// can only be right-sized once that gap is counted.
func (uc *OrchestratorUseCase) observeOutputUsage(plan *uc_interfaces.ExecutionPlan, outputTokens int64, userID uuid.UUID) {
	if plan == nil || plan.MaxOutputTokens <= 0 || outputTokens <= int64(plan.MaxOutputTokens) {
		return
	}
	tokenCeilings.onOutputOverrun(uc.logger, ceilingObservation{
		Operation:       string(plan.Operation),
		EstimatedInput:  plan.EstimatedPromptTokens,
		MaxInputTokens:  plan.MaxInputTokens,
		RequestedOutput: int(outputTokens),
		MaxOutputTokens: plan.MaxOutputTokens,
	}, userID)
}

// observeVisionUsage reports a Gemini vision call that reported no prompt
// tokens. The OpenAI-compatible branch never reports usage, so only Gemini
// silence means the meter regressed and vision cost stopped being counted.
// provider is the provider that actually served the call, not the planned one.
func (uc *OrchestratorUseCase) observeVisionUsage(operation aioperation.AIOperation, provider string, promptTokens int64, userID uuid.UUID) {
	if promptTokens > 0 || provider != providerGemini {
		return
	}
	tokenCeilings.onVisionUsageMissing(uc.logger, string(operation), userID)
}
