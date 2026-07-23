package contract

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
)

type IAICostDashboardReader interface {
	GetAdminAIUsageStats(ctx context.Context, snapshotDate time.Time) (*dto.AIUsageAdminStatsDTO, error)
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
