package usecase

import (
	"context"
	"time"

	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	identityContract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	subscriptionContract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

const microVNDDivisor = 1000000.0

type AdminDashboardUseCase struct {
	dashboardRepo      dashboardRepos.IDashboardRepository
	userReader         identityContract.IUserDashboardReader
	brandReader        brandContract.IBrandDashboardReader
	aiCostReader       aiContract.IAICostDashboardReader
	subscriptionReader subscriptionContract.ISubscriptionDashboardReader
}

func NewAdminDashboardUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	userReader identityContract.IUserDashboardReader,
	brandReader brandContract.IBrandDashboardReader,
	aiCostReader aiContract.IAICostDashboardReader,
	subscriptionReader subscriptionContract.ISubscriptionDashboardReader,
) uc_interfaces.IAdminDashboardUseCase {
	return &AdminDashboardUseCase{
		dashboardRepo:      dashboardRepo,
		userReader:         userReader,
		brandReader:        brandReader,
		aiCostReader:       aiCostReader,
		subscriptionReader: subscriptionReader,
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
		subscriptionStats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctx, now)
		if err != nil {
			return nil, err
		}

		if subscriptionStats == nil {
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
			actualCostVND := float64(aiStats.AIActualCostMicroVND) / microVNDDivisor
			res.NetAIMarginVND = -actualCostVND
		}
		res.SubscriptionRevenueVND = subscriptionStats.TotalSubscriptionRevenue.InexactFloat64()
		res.PayOSSuccessCount = subscriptionStats.PayOSSuccessCount

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
		if aiStats == nil {
			return nil, dashboarderrors.ErrDashboardDataNotFound()
		}

		actualCostVND := float64(aiStats.AIActualCostMicroVND) / microVNDDivisor
		netMargin := -actualCostVND

		return &dto.AIMarginAnalyticsRes{
			FromDate:                fromDate,
			ToDate:                  toDate,
			TotalSubscriptionRevVND: 0,
			TotalAIActualCostVND:    actualCostVND,
			NetAIMarginVND:          netMargin,
			MarginPercentage:        0.0,
			TotalAIPaidTokens:       aiStats.AIPaidTokens,
			AIRequestsCount:         int64(aiStats.AIRequestsCount),
			AIErrorCount:            int64(aiStats.AIErrorCount),
		}, nil
	}

	totalRev := 0.0
	totalMargin := 0.0
	var totalTokens int64
	var totalRequests int64
	var totalErrors int64

	for _, s := range snapshots {
		totalRev += s.SubscriptionRevenueVND
		totalMargin += s.NetAIMarginVND
		totalTokens += s.AIPaidTokens
		totalRequests += int64(s.AIRequestsCount)
		totalErrors += int64(s.AIErrorCount)
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

	if latest == nil {
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
	aiMargin, err := uc.GetAIMarginAnalytics(ctx, now.AddDate(0, 0, -dashboard_app.DefaultLookbackDays), now)
	if err != nil {
		return nil, apperror.NewInternalError("Không thể truy vấn dữ liệu biên lợi nhuận AI. Vui lòng thử lại sau.")
	}

	payos, err := uc.GetPayOSReconciliation(ctx)
	if err != nil {
		return nil, apperror.NewInternalError("Không thể truy vấn dữ liệu đối soát PayOS. Vui lòng thử lại sau.")
	}

	return &dto.AdminOverviewRes{
		KPICards:       kpi,
		AIMargin:       aiMargin,
		PayOSReconcile: payos,
	}, nil
}
