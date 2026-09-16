package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
)

type IAICostDashboardReader interface {
	GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error)
	GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]dto.AIEventDrillDownDTO, int64, error)
	GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]dto.AIUsageByOperationDTO, error)
	GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error)
	GetAIErrorBreakdown(ctx context.Context) ([]dto.AIErrorBreakdownDTO, error)
}

type AICostDashboardReader struct {
	costUC uc_interfaces.IAICostUseCase
}

func NewAICostDashboardReader(costUC uc_interfaces.IAICostUseCase) IAICostDashboardReader {
	return &AICostDashboardReader{costUC: costUC}
}

func (r *AICostDashboardReader) GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error) {
	return r.costUC.GetAdminAIUsageStats(ctx, snapshotDate)
}

func (r *AICostDashboardReader) GetAIEventsDrillDown(ctx context.Context, status string, page, limit int) ([]dto.AIEventDrillDownDTO, int64, error) {
	return r.costUC.GetAIEventsDrillDown(ctx, status, page, limit)
}

func (r *AICostDashboardReader) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) ([]dto.AIUsageByOperationDTO, error) {
	return r.costUC.GetAIUsageByOperation(ctx, fromDate, toDate)
}

func (r *AICostDashboardReader) GetTotalAIRequestCount(ctx context.Context, fromDate, toDate time.Time) (int64, error) {
	return r.costUC.GetTotalAIRequestCount(ctx, fromDate, toDate)
}

func (r *AICostDashboardReader) GetAIErrorBreakdown(ctx context.Context) ([]dto.AIErrorBreakdownDTO, error) {
	return r.costUC.GetAIErrorBreakdown(ctx)
}
