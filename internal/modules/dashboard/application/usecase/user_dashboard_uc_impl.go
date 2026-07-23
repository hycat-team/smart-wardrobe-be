package usecase

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	wardrobeContract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

	"github.com/google/uuid"
)

type UserDashboardUseCase struct {
	dashboardRepo  dashboardRepos.IDashboardRepository
	wardrobeReader wardrobeContract.IWardrobeDashboardReader
}

func NewUserDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	wardrobeReader wardrobeContract.IWardrobeDashboardReader,
) uc_interfaces.IUserDashboardUseCase {
	return &UserDashboardUseCase{
		dashboardRepo:  dashboardRepo,
		wardrobeReader: wardrobeReader,
	}
}

func (uc *UserDashboardUseCase) GetWardrobeInsights(ctx context.Context, userID uuid.UUID) (*dto.UserWardrobeInsightsRes, error) {
	summary, err := uc.dashboardRepo.GetUserWardrobeSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalItems := 0
	totalValue := 0.0
	avgCPW := 0.0
	underutilizedCount := 0
	categoryDist := map[string]int{}
	underutilizedItems := make([]dto.UnderutilizedItemDTO, 0)

	if summary != nil {
		totalItems = summary.TotalItems
		totalValue = summary.TotalWardrobeValueVND
		avgCPW = summary.AverageCostPerWearVND
		underutilizedCount = summary.UnderutilizedItemsCount
		if len(summary.CategoryDistribution) > 0 {
			_ = json.Unmarshal([]byte(summary.CategoryDistribution), &categoryDist)
		}
		if len(summary.UnderutilizedItems) > 0 {
			_ = json.Unmarshal([]byte(summary.UnderutilizedItems), &underutilizedItems)
		}
	}

	return &dto.UserWardrobeInsightsRes{
		TotalItems:              totalItems,
		TotalWardrobeValueVND:   totalValue,
		AverageCostPerWearVND:   avgCPW,
		UnderutilizedItemsCount: underutilizedCount,
		CategoryDistribution:    categoryDist,
		UnderutilizedItems:      underutilizedItems,
	}, nil
}
