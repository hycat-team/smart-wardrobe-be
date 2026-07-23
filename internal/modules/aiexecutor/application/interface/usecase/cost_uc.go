package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aioperation"

	"github.com/google/uuid"
)

type IAICostUseCase interface {
	Prepare(ctx context.Context, plan *dto.ExecutionPlan) (*dto.AICostDecision, error)
	PrepareFree(ctx context.Context, requestID uuid.UUID, userID uuid.UUID, operation aioperation.AIOperation, promptTokens int64, maxOutputTokens int) (*dto.AICostDecision, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID) error
	Confirm(ctx context.Context, requestID uuid.UUID, usage dto.AIUsage) error
	Release(ctx context.Context, requestID uuid.UUID, reason string) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
	GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error)
}
