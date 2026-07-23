package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"

	"github.com/google/uuid"
)

type IBrandDashboardUseCase interface {
	GetBrandKPICards(ctx context.Context, brandID uuid.UUID) (*dto.BrandKPICardsRes, error)
	GetPointsLiability(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) (*dto.BrandPointsLiabilityRes, error)
	GetDigitalSampleLabAnalytics(ctx context.Context, brandID uuid.UUID, itemID uuid.UUID) (*dto.DigitalSampleLabAnalyticsRes, error)
}
