package usecase

import (
	"context"
	"time"

	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	identityContract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
)

type AdminDashboardUseCase struct {
	dashboardRepo dashboardRepos.IDashboardRepository
	userReader    identityContract.IUserDashboardReader
	brandReader   brandContract.IBrandDashboardReader
	aiCostReader  aiContract.IAICostDashboardReader
}

func NewAdminDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	userReader identityContract.IUserDashboardReader,
	brandReader brandContract.IBrandDashboardReader,
	aiCostReader aiContract.IAICostDashboardReader,
) uc_interfaces.IAdminDashboardUseCase {
	return &AdminDashboardUseCase{
		dashboardRepo: dashboardRepo,
		userReader:    userReader,
		brandReader:   brandReader,
		aiCostReader:  aiCostReader,
	}
}

func (uc *AdminDashboardUseCase) GetAdminKPICards(ctx context.Context) (*dto.AdminKPICardsRes, error) {
	latest, err := uc.dashboardRepo.GetLatestAdminSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	if latest == nil {
		now := time.Now()
		userStats, err := uc.userReader.GetAdminUserStats(ctx, now)
		if err != nil {
			return nil, err
		}
		brandStats, err := uc.brandReader.GetAdminBrandStats(ctx, now)
		if err != nil {
			return nil, err
		}
		aiStats, err := uc.aiCostReader.GetAdminAIUsageStats(ctx, now)
		if err != nil {
			return nil, err
		}

		if userStats == nil && brandStats == nil && aiStats == nil {
			return nil, dashboarderrors.ErrDashboardDataNotFound()
		}

		res := &dto.AdminKPICardsRes{
			UpdatedAt: now,
		}
		if userStats != nil {
			res.TotalUsers = userStats.TotalUsers
			res.ActiveUsersDAU = userStats.ActiveUsersDau
		}
		if brandStats != nil {
			res.TotalBrands = brandStats.TotalBrands
			res.PendingReviewBrands = brandStats.PendingReviewBrands
		}
		if aiStats != nil {
			actualCostVND := float64(aiStats.AIActualCostMicroVND) / 1000000.0
			res.NetAIMarginVND = -actualCostVND
		}
		return res, nil
	}

	return &dto.AdminKPICardsRes{
		TotalUsers:             latest.TotalUsers,
		ActiveUsersDAU:         latest.ActiveUsersDau,
		TotalBrands:            latest.TotalBrands,
		PendingReviewBrands:    latest.PendingReviewBrands,
		SubscriptionRevenueVND: latest.SubscriptionRevenueVND,
		NetAIMarginVND:         latest.NetAIMarginVND,
		PayOSSuccessCount:      latest.PayOSSuccessCount,
		UpdatedAt:              latest.UpdatedAt,
	}, nil
}

func (uc *AdminDashboardUseCase) GetAIMarginAnalytics(ctx context.Context, fromDate, toDate time.Time) (*dto.AIMarginAnalyticsRes, error) {
	if fromDate.IsZero() {
		fromDate = time.Now().AddDate(0, 0, -30)
	}
	if toDate.IsZero() {
		toDate = time.Now()
	}

	snapshots, err := uc.dashboardRepo.GetAdminSnapshots(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		// Fallback to live AI reader
		aiStats, err := uc.aiCostReader.GetAdminAIUsageStats(ctx, toDate)
		if err != nil {
			return nil, err
		}
		if aiStats == nil || (aiStats.AIRequestsCount == 0 && aiStats.AIActualCostMicroVND == 0) {
			return nil, dashboarderrors.ErrDashboardDataNotFound()
		}

		actualCostVND := float64(aiStats.AIActualCostMicroVND) / 1000000.0
		netMargin := -actualCostVND

		return &dto.AIMarginAnalyticsRes{
			FromDate:                fromDate,
			ToDate:                  toDate,
			TotalSubscriptionRevVND: 0,
			TotalAIActualCostVND:    actualCostVND,
			NetAIMarginVND:          netMargin,
			MarginPercentage:        0.0,
			TotalAIPaidTokens:       aiStats.AIPaidTokens,
			AIRequestsCount:         aiStats.AIRequestsCount,
			AIErrorCount:            aiStats.AIErrorCount,
		}, nil
	}

	totalRev := 0.0
	totalMargin := 0.0
	var totalTokens int64
	totalRequests := 0
	totalErrors := 0

	for _, s := range snapshots {
		totalRev += s.SubscriptionRevenueVND
		totalMargin += s.NetAIMarginVND
		totalTokens += s.AIPaidTokens
		totalRequests += s.AIRequestsCount
		totalErrors += s.AIErrorCount
	}

	actualCostVND := totalRev - totalMargin

	marginPct := 0.0
	if totalRev > 0 {
		marginPct = (totalMargin / totalRev) * 100
	}

	return &dto.AIMarginAnalyticsRes{
		FromDate:                fromDate,
		ToDate:                  toDate,
		TotalSubscriptionRevVND: totalRev,
		TotalAIActualCostVND:    actualCostVND,
		NetAIMarginVND:          totalMargin,
		MarginPercentage:        marginPct,
		TotalAIPaidTokens:       totalTokens,
		AIRequestsCount:         totalRequests,
		AIErrorCount:            totalErrors,
	}, nil
}

func (uc *AdminDashboardUseCase) GetPayOSReconciliation(ctx context.Context) (*dto.PayOSReconciliationRes, error) {
	latest, err := uc.dashboardRepo.GetLatestAdminSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	if latest == nil || latest.PayOSTransactionCount == 0 {
		return nil, dashboarderrors.ErrDashboardDataNotFound()
	}

	failed := max(latest.PayOSTransactionCount-latest.PayOSSuccessCount, 0)

	return &dto.PayOSReconciliationRes{
		TotalTransactions: latest.PayOSTransactionCount,
		SuccessCount:      latest.PayOSSuccessCount,
		FailedCount:       failed,
		TotalSuccessValue: latest.SubscriptionRevenueVND,
	}, nil
}

func (uc *AdminDashboardUseCase) GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error) {
	kpi, err := uc.GetAdminKPICards(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	aiMargin, err := uc.GetAIMarginAnalytics(ctx, now.AddDate(0, 0, -30), now)
	if err != nil {
		aiMargin = nil
	}

	payos, err := uc.GetPayOSReconciliation(ctx)
	if err != nil {
		payos = nil
	}

	return &dto.AdminOverviewRes{
		KPICards:       kpi,
		AIMargin:       aiMargin,
		PayOSReconcile: payos,
	}, nil
}
