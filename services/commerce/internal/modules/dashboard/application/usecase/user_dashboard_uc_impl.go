package usecase

import (
	"context"
	"encoding/json"
	"time"

	brandContract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	dashboard_app "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/entities"
	dashboardRepos "smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/repositories"
	fashionContract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"
	wardrobeContract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"

	"github.com/google/uuid"
)

type UserDashboardUseCase struct {
	dashboardRepo  dashboardRepos.IDashboardRepository
	wardrobeReader wardrobeContract.IWardrobeDashboardReader
	brandReader    brandContract.IBrandDashboardReader
	fashionReader  fashionContract.IFashionDashboardReader
}

func NewUserDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	wardrobeReader wardrobeContract.IWardrobeDashboardReader,
	brandReader brandContract.IBrandDashboardReader,
	fashionReader fashionContract.IFashionDashboardReader,
) uc_interfaces.IUserDashboardUseCase {
	return &UserDashboardUseCase{
		dashboardRepo:  dashboardRepo,
		wardrobeReader: wardrobeReader,
		brandReader:    brandReader,
		fashionReader:  fashionReader,
	}
}

func (uc *UserDashboardUseCase) GetWardrobeCategoryDistribution(ctx context.Context, userID uuid.UUID) (*dto.WardrobeCategoryDistributionRes, error) {
	fashionItemIDs, err := uc.wardrobeReader.GetUserWardrobeFashionItemIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(fashionItemIDs) == 0 {
		return &dto.WardrobeCategoryDistributionRes{TotalItems: 0, Categories: []dto.CategoryDistributionDTO{}}, nil
	}

	itemCategoryMap := map[uuid.UUID]*uuid.UUID{}
	categoryIDs := []uuid.UUID{}
	seenCategory := map[uuid.UUID]bool{}
	if len(fashionItemIDs) > 0 {
		mappings, err := uc.fashionReader.GetCategoryIDsByFashionItemIDs(ctx, fashionItemIDs)
		if err != nil {
			return nil, err
		}
		for _, m := range mappings {
			itemCategoryMap[m.FashionItemID] = m.CategoryID
			if m.CategoryID != nil && !seenCategory[*m.CategoryID] {
				seenCategory[*m.CategoryID] = true
				categoryIDs = append(categoryIDs, *m.CategoryID)
			}
		}
	}

	categoryMap := map[uuid.UUID]string{}
	if len(categoryIDs) > 0 {
		categories, err := uc.fashionReader.GetCategoriesByIDs(ctx, categoryIDs)
		if err != nil {
			return nil, err
		}
		for _, c := range categories {
			categoryMap[c.ID] = c.Name
		}
	}

	counts := map[uuid.UUID]int{}
	for _, fashionItemID := range fashionItemIDs {
		categoryID := itemCategoryMap[fashionItemID]
		if categoryID != nil {
			counts[*categoryID]++
		}
	}

	totalItems := 0
	categories := make([]dto.CategoryDistributionDTO, 0, len(counts))
	for categoryID, itemCount := range counts {
		totalItems += itemCount
		categories = append(categories, dto.CategoryDistributionDTO{
			CategoryID:   categoryID,
			CategoryName: categoryMap[categoryID],
			ItemCount:    itemCount,
			Percentage:   0,
		})
	}

	for i := range categories {
		if totalItems > 0 {
			categories[i].Percentage = float64(categories[i].ItemCount) / float64(totalItems) * 100
		}
	}

	return &dto.WardrobeCategoryDistributionRes{
		TotalItems: totalItems,
		Categories: categories,
	}, nil
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
			UnderutilizedItemsCount: summary.UnderutilizedItemsCount,
			UnderutilizedItems:      underutilizedItems,
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

	// Refresh expiring vouchers count from brand reader (live data)
	expiringVouchers, err := uc.brandReader.GetUserExpiringBenefitsCount(ctx, userID)
	if err != nil {
		return nil, err
	}

	newSummary := &entities.DashboardUserWardrobeSummary{
		UserID:                  userID,
		TotalItems:              totalItems,
		TotalWardrobeValueVND:   totalValue,
		AverageCostPerWearVND:   avgCPW,
		UnderutilizedItemsCount: underutilizedCount,
		UnderutilizedItems:      underutilizedJSON,
		ExpiringVouchersCount:   expiringVouchers,
		UpdatedAt:               time.Now(),
	}
	_ = uc.dashboardRepo.UpsertUserWardrobeSummary(ctx, newSummary)

	return &dto.UserWardrobeInsightsRes{
		TotalItems:              totalItems,
		TotalWardrobeValueVND:   totalValue,
		UnderutilizedItemsCount: underutilizedCount,
		UnderutilizedItems:      underutilizedItems,
		ExpiringVouchersCount:   expiringVouchers,
	}, nil
}
