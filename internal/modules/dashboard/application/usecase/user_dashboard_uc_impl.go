package usecase

import (
	"context"
	"encoding/json"
	"time"

	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
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

	// 1. Cache Hit: Read from snapshot table if fresh (TTL = 1 hour)
	if summary != nil && time.Since(summary.UpdatedAt) < dashboard_app.DashboardCacheTTL {
		underutilizedItems := make([]dto.UnderutilizedItemDTO, 0)
		if len(summary.UnderutilizedItems) > 0 {
			_ = json.Unmarshal([]byte(summary.UnderutilizedItems), &underutilizedItems)
		}
		return &dto.UserWardrobeInsightsRes{
			TotalItems:              summary.TotalItems,
			TotalWardrobeValueVND:   summary.TotalWardrobeValueVND,
			AverageItemValueVND:     summary.AverageCostPerWearVND,
			UnderutilizedItemsCount: summary.UnderutilizedItemsCount,
			UnderutilizedItems:      underutilizedItems,
			AIQuotaRemaining:        summary.AIQuotaRemaining,
			ExpiringVouchersCount:   summary.ExpiringVouchersCount,
		}, nil
	}

	// 2. Cache Miss / Stale: Calculate on-demand via wardrobeReader
	stats, err := uc.wardrobeReader.GetUserWardrobeSummaryStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	realItems, err := uc.wardrobeReader.GetUnderutilizedItems(ctx, userID, dashboard_app.UnderutilizedItemsLimit)
	if err != nil {
		return nil, err
	}

	if (stats == nil || stats.TotalItems == 0) && len(realItems) == 0 {
		return nil, dashboarderrors.ErrDashboardDataNotFound()
	}

	totalItems := 0
	totalValue := 0.0
	avgCPW := 0.0
	underutilizedCount := 0
	if stats != nil {
		totalItems = stats.TotalItems
		totalValue = stats.TotalWardrobeValueVND
		avgCPW = stats.AverageCostPerWearVND
		underutilizedCount = stats.UnderutilizedItemsCount
	}

	underutilizedItems := make([]dto.UnderutilizedItemDTO, 0, len(realItems))
	for _, item := range realItems {
		underutilizedItems = append(underutilizedItems, dto.UnderutilizedItemDTO{
			ItemID:           item.ItemID,
			Name:             item.Name,
			CategoryName:     item.CategoryName,
			PurchasePriceVND: item.PurchasePriceVND,
			LastWornDaysAgo:  item.LastWornDaysAgo,
			ImageURL:         item.ImageURL,
		})
	}

	// 3. Write-Back: Upsert fresh snapshot to database for future fast reads
	underutilizedJSON, err := json.Marshal(underutilizedItems)
	if err != nil {
		underutilizedJSON = []byte("[]")
	}

	// Read existing AI quota & voucher counts from stale snapshot if available
	aiQuota := 0
	expiringVouchers := 0
	if summary != nil {
		aiQuota = summary.AIQuotaRemaining
		expiringVouchers = summary.ExpiringVouchersCount
	}

	newSummary := &entities.DashboardUserWardrobeSummary{
		UserID:                  userID,
		TotalItems:              totalItems,
		TotalWardrobeValueVND:   totalValue,
		AverageCostPerWearVND:   avgCPW,
		UnderutilizedItemsCount: underutilizedCount,
		UnderutilizedItems:      underutilizedJSON,
		AIQuotaRemaining:        aiQuota,
		ExpiringVouchersCount:   expiringVouchers,
		UpdatedAt:               time.Now(),
	}
	_ = uc.dashboardRepo.UpsertUserWardrobeSummary(ctx, newSummary)

	return &dto.UserWardrobeInsightsRes{
		TotalItems:              totalItems,
		TotalWardrobeValueVND:   totalValue,
		AverageItemValueVND:     avgCPW,
		UnderutilizedItemsCount: underutilizedCount,
		UnderutilizedItems:      underutilizedItems,
		AIQuotaRemaining:        aiQuota,
		ExpiringVouchersCount:   expiringVouchers,
	}, nil
}
