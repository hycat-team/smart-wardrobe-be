package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"

	"github.com/google/uuid"
)

// IAICostAttemptUseCase is the attempt-aware admission half of the cost use
// case introduced by feature 032. IAICostUseCase embeds it so the orchestrator
// can call admission and cost on the same injected dependency.
type IAICostAttemptUseCase interface {
	AdmitUnbilledAttempt(ctx context.Context, admission UnbilledAttemptAdmission) (*AICostDecision, error)
	PrepareBillableAttempt(ctx context.Context, admission BillableAttemptAdmission) (*AICostDecision, error)
}

type IAICostUseCase interface {
	IAICostAttemptUseCase
	Prepare(ctx context.Context, plan *ExecutionPlan) (*AICostDecision, error)
	PrepareFree(ctx context.Context, requestID uuid.UUID, userID uuid.UUID, operation aioperation.AIOperation, promptTokens int64, maxOutputTokens int) (*AICostDecision, error)
	MarkInFlight(ctx context.Context, requestID uuid.UUID) error
	Confirm(ctx context.Context, requestID uuid.UUID, usage AIUsage) error
	Release(ctx context.Context, requestID uuid.UUID, reason string) error
	MarkUnknown(ctx context.Context, requestID uuid.UUID, reason string) error
	ExpireUnknown(ctx context.Context, now time.Time, limit int) (int64, error)
	GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error)
	GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]dto.AIEventDrillDownDTO, int64, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]dto.AIUsageByOperationDTO, error)
	GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error)
	GetAIErrorBreakdown(ctx context.Context, from, to time.Time) ([]dto.AIErrorBreakdownDTO, error)
}
