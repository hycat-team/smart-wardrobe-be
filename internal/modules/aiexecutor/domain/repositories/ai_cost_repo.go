package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
)

type IAICostRepository interface {
	ResolvePolicy(ctx context.Context, subscriptionPlanID uuid.UUID, userID uuid.UUID, operation aioperation.AIOperation, now time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error)
	Reserve(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant, policy *aiexecutor_entities.AICostPolicy, operation *aiexecutor_entities.AICostPolicyOperation, event *aiexecutor_entities.AIUsageEvent, now time.Time) (*aiexecutor_entities.AIUsagePeriodLedger, bool, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID, at time.Time) error
	Confirm(ctx context.Context, requestID uuid.UUID, promptTokens, outputTokens, thinkingTokens, actualCost int64, provider, model, finishReason string, at time.Time) error
	Release(ctx context.Context, requestID uuid.UUID, reason string, at time.Time) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string, expiresAt time.Time) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
	ExpireGrantsByUserID(ctx context.Context, userID uuid.UUID, now time.Time) error
	CreateGrant(ctx context.Context, grant *aiexecutor_entities.AIPolicyGrant) error
	CreateEvent(ctx context.Context, event *aiexecutor_entities.AIUsageEvent) error
}
