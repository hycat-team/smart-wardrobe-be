package usecase

import (
	"context"
	"time"

	aiContract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	dashboard_app "smart-wardrobe-be/internal/modules/dashboard/application"
	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	identityContract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	subscriptionContract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/pkg/utils/numberutils"
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
		if subscriptionStats != nil {
			res.SubscriptionRevenueVND = subscriptionStats.TotalSubscriptionRevenue.InexactFloat64()
			res.PayOSSuccessCount = subscriptionStats.PayOSSuccessCount
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
		if aiStats != nil {
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

		return &dto.AIMarginAnalyticsRes{
			FromDate:                fromDate,
			ToDate:                  toDate,
			TotalSubscriptionRevVND: 0,
			TotalAIActualCostVND:    0,
			NetAIMarginVND:          0,
			MarginPercentage:        0.0,
			TotalAIPaidTokens:       0,
			AIRequestsCount:         0,
			AIErrorCount:            0,
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
		return &dto.PayOSReconciliationRes{
			TotalTransactions: 0,
			SuccessCount:      0,
			FailedCount:       0,
			TotalSuccessValue: 0,
		}, nil
	}

	failed := max(latest.PayOSTransactionCount-latest.PayOSSuccessCount, 0)

	return &dto.PayOSReconciliationRes{
		TotalTransactions: latest.PayOSTransactionCount,
		SuccessCount:      latest.PayOSSuccessCount,
		FailedCount:       failed,
		TotalSuccessValue: latest.SubscriptionRevenueVND,
	}, nil
}

func (uc *AdminDashboardUseCase) GetSubscriptionDistribution(ctx context.Context) (*dto.SubscriptionDistributionRes, error) {
	raw, err := uc.dashboardRepo.GetSubscriptionDistribution(ctx)
	if err != nil {
		return nil, err
	}

	totalUsers := 0
	distribution := make([]dto.SubscriptionDistributionDTO, 0, len(raw))
	for _, r := range raw {
		planCode, _ := r["plan_code"].(string)
		userCount, _ := r["user_count"].(int64)
		totalUsers += int(userCount)
		distribution = append(distribution, dto.SubscriptionDistributionDTO{
			PlanCode:   planCode,
			UserCount:  int(userCount),
			Percentage: 0,
		})
	}

	for i := range distribution {
		if totalUsers > 0 {
			distribution[i].Percentage = float64(distribution[i].UserCount) / float64(totalUsers) * 100
		}
	}

	return &dto.SubscriptionDistributionRes{
		Distribution: distribution,
		TotalUsers:   totalUsers,
	}, nil
}

func (uc *AdminDashboardUseCase) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) (*dto.RevenueBreakdownRes, error) {
	if fromDate.IsZero() {
		fromDate = time.Now().AddDate(0, -6, 0)
	}
	if toDate.IsZero() {
		toDate = time.Now()
	}

	raw, err := uc.dashboardRepo.GetRevenueBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	totalRevenue := 0.0
	breakdown := make([]dto.RevenueBreakdownDTO, 0, len(raw))
	for _, r := range raw {
		planCode, _ := r["plan_code"].(string)
		month, _ := r["month"].(time.Time)
		txnCount, _ := r["transaction_count"].(int64)
		revenue := numberutils.ToFloat64(r["revenue"])
		totalRevenue += revenue
		breakdown = append(breakdown, dto.RevenueBreakdownDTO{
			PlanCode:         planCode,
			Month:            month.Format("2006-01"),
			TransactionCount: int(txnCount),
			RevenueVND:       revenue,
		})
	}

	return &dto.RevenueBreakdownRes{
		FromDate:        fromDate.Format("2006-01-02"),
		ToDate:          toDate.Format("2006-01-02"),
		Breakdown:       breakdown,
		TotalRevenueVND: totalRevenue,
	}, nil
}

func (uc *AdminDashboardUseCase) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) (*dto.AIUsageByOperationRes, error) {
	if fromDate.IsZero() {
		fromDate = time.Now().AddDate(0, 0, -30)
	}
	if toDate.IsZero() {
		toDate = time.Now()
	}

	raw, err := uc.dashboardRepo.GetAIUsageByOperation(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	operations := make([]dto.AIUsageByOperationDTO, 0, len(raw))
	for _, r := range raw {
		op, _ := r["operation"].(string)
		route, _ := r["logical_route"].(string)
		reqCount, _ := r["request_count"].(int64)

		var costMicro, promptTokens, outputTokens float64
		if v, ok := r["total_cost_micro_vnd"]; ok {
			costMicro = numberutils.ToFloat64(v)
		}
		if v, ok := r["total_prompt_tokens"]; ok {
			promptTokens = numberutils.ToFloat64(v)
		}
		if v, ok := r["total_output_tokens"]; ok {
			outputTokens = numberutils.ToFloat64(v)
		}
		operations = append(operations, dto.AIUsageByOperationDTO{
			Operation:         op,
			LogicalRoute:      route,
			RequestCount:      reqCount,
			TotalCostVND:      costMicro / microVNDDivisor,
			TotalPromptTokens: int64(promptTokens),
			TotalOutputTokens: int64(outputTokens),
		})
	}

	return &dto.AIUsageByOperationRes{
		FromDate:   fromDate.Format("2006-01-02"),
		ToDate:     toDate.Format("2006-01-02"),
		Operations: operations,
	}, nil
}

func (uc *AdminDashboardUseCase) GetAIErrorBreakdown(ctx context.Context) (*dto.AIErrorBreakdownRes, error) {
	raw, err := uc.dashboardRepo.GetAIErrorBreakdown(ctx)
	if err != nil {
		return nil, err
	}

	totalErrors := int64(0)
	breakdown := make([]dto.AIErrorBreakdownDTO, 0, len(raw))
	for _, r := range raw {
		errorCode, _ := r["error_code"].(string)
		provider, _ := r["provider"].(string)
		model, _ := r["model"].(string)
		errCount, _ := r["error_count"].(int64)
		totalErrors += errCount
		breakdown = append(breakdown, dto.AIErrorBreakdownDTO{
			ErrorCode:  errorCode,
			Provider:   provider,
			Model:      model,
			ErrorCount: errCount,
		})
	}

	// Query total requests in last 30 days for error rate calculation
	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -30)
	totalRequests, _ := uc.dashboardRepo.GetTotalAIRequestCount(ctx, fromDate, toDate)

	errorRate := 0.0
	if totalRequests > 0 {
		errorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	return &dto.AIErrorBreakdownRes{
		TotalErrors:   totalErrors,
		TotalRequests: totalRequests,
		ErrorRate:     errorRate,
		Breakdown:     breakdown,
	}, nil
}

func (uc *AdminDashboardUseCase) GetRenewalStats(ctx context.Context) (*dto.RenewalStatsRes, error) {
	raw, err := uc.dashboardRepo.GetRenewalStats(ctx)
	if err != nil {
		return nil, err
	}

	totalAttempts := int(numberutils.ToFloat64(raw["total_attempts"]))
	successCount := int(numberutils.ToFloat64(raw["success_count"]))
	failedCount := int(numberutils.ToFloat64(raw["failed_count"]))
	avgRetry := numberutils.ToFloat64(raw["avg_retry_count"])

	successRate := 0.0
	if totalAttempts > 0 {
		successRate = float64(successCount) / float64(totalAttempts) * 100
	}

	return &dto.RenewalStatsRes{
		TotalAttempts: int(totalAttempts),
		SuccessCount:  int(successCount),
		FailedCount:   int(failedCount),
		SuccessRate:   successRate,
		AvgRetryCount: avgRetry,
	}, nil
}

func (uc *AdminDashboardUseCase) GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error) {
	kpi, err := uc.GetAdminKPICards(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	aiMargin, _ := uc.GetAIMarginAnalytics(ctx, now.AddDate(0, 0, -dashboard_app.DefaultLookbackDays), now)

	payos, _ := uc.GetPayOSReconciliation(ctx)

	return &dto.AdminOverviewRes{
		KPICards:       kpi,
		AIMargin:       aiMargin,
		PayOSReconcile: payos,
	}, nil
}
