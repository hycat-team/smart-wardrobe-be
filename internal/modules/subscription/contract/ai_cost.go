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
)

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
	Prepare(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64) (*AICostDecision, error)
	PrepareFree(ctx context.Context, userID uuid.UUID, operation string, promptTokens int64, maxOutputTokens int) (*AICostDecision, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID) error
	Confirm(ctx context.Context, requestID uuid.UUID, usage AIUsage) error
	Release(ctx context.Context, requestID uuid.UUID, reason string) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
}
