package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	AIOperationChat     = "chat"
	AIOperationOutfit   = "outfit"
	AIOperationSummary  = "summary"
	AIOperationRewriter = "rewriter"
	AIRoutePaid         = "paid"
	AIRouteFree         = "free"
	AIRouteLocal        = "local"

	TokenEstimationLocal         = "LOCAL"
	TokenEstimationProviderCount = "PROVIDER_COUNT"
)

type AITokenEstimationMeta struct {
	EstimatedPromptTokens int64  // Total input tokens estimated or provider-counted during preflight
	TokenEstimationMethod string // Method used: 'LOCAL' or 'PROVIDER_COUNT'
	TokenCountLatencyMs   *int64 // Latency of countTokens call in ms. nil for local path.
}

type AICostDecision struct {
	RequestID       uuid.UUID
	Route           string
	MaxInputTokens  int
	MaxOutputTokens int
	Tracked         bool
}

type AIUsage struct {
	PromptTokens   int64
	OutputTokens   int64
	ThinkingTokens int64
	FinishReason   string
	Provider       string
	Model          string
}

type IAICostPolicyContract interface {
	Prepare(ctx context.Context, userID uuid.UUID, operation string, meta AITokenEstimationMeta) (*AICostDecision, error)
	PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*AICostDecision, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID) error
	Confirm(ctx context.Context, requestID uuid.UUID, usage AIUsage) error
	Release(ctx context.Context, requestID uuid.UUID, reason string) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
}
