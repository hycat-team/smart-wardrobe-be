package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
)

type IAdminStatsUseCase interface {
	GetAdminSubscriptionStats(ctx context.Context, snapshotDate time.Time) (*dto.AdminSubscriptionStatsDTO, error)
}
