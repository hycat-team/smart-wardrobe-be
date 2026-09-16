package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"

	"github.com/google/uuid"
)

type IUserDashboardUseCase interface {
	GetWardrobeInsights(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeInsightsRes, error)

	// Expansion APIs
	GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) (*dto.WardrobeCategoryDistributionRes, error)
}
