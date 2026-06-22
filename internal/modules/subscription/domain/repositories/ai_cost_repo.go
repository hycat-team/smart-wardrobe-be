package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type IAICostRepository interface {
	ResolvePolicy(ctx context.Context, userID uuid.UUID, operation string, now time.Time) (*entities.UserAIPolicyGrant, *entities.AICostPolicy, *entities.AICostPolicyOperation, error)
	Reserve(ctx context.Context, grant *entities.UserAIPolicyGrant, policy *entities.AICostPolicy, operation *entities.AICostPolicyOperation, event *entities.AIUsageEvent, now time.Time) (*entities.AIUsagePeriodLedger, bool, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID, at time.Time) error
	Confirm(ctx context.Context, requestID uuid.UUID, promptTokens, outputTokens, thinkingTokens, actualCost int64, provider, model, finishReason string, at time.Time) error
	Release(ctx context.Context, requestID uuid.UUID, reason string, at time.Time) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string, expiresAt time.Time) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
}
