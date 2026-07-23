package usecase

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
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
	} else {
		stats, err := uc.wardrobeReader.GetUserWardrobeSummaryStats(ctx, userID)
		if err != nil {
			return nil, err
		}
		if stats != nil {
			totalItems = stats.TotalItems
			totalValue = stats.TotalWardrobeValueVND
			avgCPW = stats.AverageCostPerWearVND
			underutilizedCount = stats.UnderutilizedItemsCount
		}

		realItems, err := uc.wardrobeReader.GetUnderutilizedItems(ctx, userID, 10)
		if err != nil {
			return nil, err
		}
		for _, item := range realItems {
			underutilizedItems = append(underutilizedItems, dto.UnderutilizedItemDTO{
				ItemID:           item.ItemID,
				Name:             item.Name,
				CategoryName:     item.CategoryName,
				PurchasePriceVND: item.PurchasePriceVND,
				WearCount:        item.WearCount,
				CostPerWearVND:   item.CostPerWearVND,
				LastWornDaysAgo:  item.LastWornDaysAgo,
				ImageURL:         item.ImageURL,
			})
			if item.CategoryName != "" {
				categoryDist[item.CategoryName]++
			}
		}

		if (stats == nil || stats.TotalItems == 0) && len(realItems) == 0 {
			return nil, dashboarderrors.ErrDashboardDataNotFound()
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
