package usecase

import (
	"context"
	"time"

	identityContract "smart-wardrobe-be/services/account/contract/identity"
	subscriptionContract "smart-wardrobe-be/services/account/contract/subscription"
	aiContract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	brandContract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	dashboard_app "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application"
	"smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/services/commerce/internal/modules/dashboard/domain/repositories"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"
)

const microVNDDivisor = 1000000.0

// vnInclusiveDayLabel renders one instant of a VN calendar day as the
// YYYY-MM-DD label the period responses echo. Callers step the exclusive upper
// bound (EndOfDayInVN = 00:00 of the NEXT day) back by a nanosecond first so
// the label never shows the day after the period ends.
func vnInclusiveDayLabel(t time.Time) string {
	return t.In(timeutils.VnLoc()).Format("2006-01-02")
}

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
		now := timeutils.NowVN()
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
		// The subscription stats query takes the half-open VN day pair
		// (029 R1); VNDayRange(now) is exactly the day range the repository
		// used to derive internally, so the KPI fallback keeps its behaviour.
		statsFrom, statsTo := timeutils.VNDayRange(now)
		subscriptionStats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctx, statsFrom, statsTo)
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
	// Defaults are VN calendar-day boundaries now (029 R1/FR-012): the range
	// meaning is the same 30 days it was, the edges no longer drift with the
	// clock or the server timezone.
	if fromDate.IsZero() {
		fromDate = timeutils.StartOfDayInVN(timeutils.NowVN().AddDate(0, 0, -30))
	}
	if toDate.IsZero() {
		toDate = timeutils.EndOfDayInVN(timeutils.NowVN())
	}

	// The response echoes the VN calendar days, not the exclusive bounds:
	// 00:00 of the first day and 00:00 of the last day. Echoing the raw bounds
	// would label the period with 00:00 of the day AFTER the last one.
	labelFrom := timeutils.StartOfDayInVN(fromDate)
	labelTo := timeutils.StartOfDayInVN(toDate.Add(-time.Nanosecond))

	// `snapshot_date` is a DATE column and the repository compares it with `<=`,
	// while `toDate` is the EXCLUSIVE upper bound (00:00 of the next VN day):
	// passing the bound unchanged would also count the snapshot row of the day
	// AFTER the period, because that DATE value's stored 00:00 equals the bound
	// (off-by-one reproduced on the dev DB: range 14–15/10 picked up the 16/10
	// row). Step back just inside the last selected day so the window covers
	// exactly [fromDate, last day]. `GetSystemOverview` passes a mid-day `now`;
	// stepping back 1ns stays within that day, so its behaviour is unchanged.
	snapshots, err := uc.dashboardRepo.GetAdminSnapshots(ctx, fromDate, toDate.Add(-time.Nanosecond))
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return uc.aiMarginFromLiveRange(ctx, fromDate, toDate, labelFrom, labelTo)
	}

	totalRev := 0.0
	var totalCostMicro int64
	var totalTokens int64
	var totalRequests int64
	var totalErrors int64

	for _, s := range snapshots {
		totalRev += s.SubscriptionRevenueVND
		totalCostMicro += s.AIActualCostMicroVND
		totalTokens += s.AIPaidTokens
		totalRequests += int64(s.AIRequestsCount)
		totalErrors += int64(s.AIErrorCount)
	}

	// The cost column is the source of truth. This used to be derived as
	// `totalRev - totalMargin`, which inherited any drift in the stored margin
	// and made the panel disagree with `usage-by-operation` - that endpoint
	// sums the same ledger column live. Summing the column keeps both panels on
	// one number; the margin is recomputed from it so the two fields cannot
	// contradict each other.
	actualCostVND := float64(totalCostMicro) / microVNDDivisor
	totalMargin := totalRev - actualCostVND

	marginPct := 0.0
	if totalRev > 0 {
		marginPct = (totalMargin / totalRev) * 100
	}

	return &dto.AIMarginAnalyticsRes{
		FromDate:                labelFrom,
		ToDate:                  labelTo,
		TotalSubscriptionRevVND: totalRev,
		TotalAIActualCostVND:    actualCostVND,
		NetAIMarginVND:          totalMargin,
		MarginPercentage:        marginPct,
		TotalAIPaidTokens:       totalTokens,
		AIRequestsCount:         totalRequests,
		AIErrorCount:            totalErrors,
	}, nil
}

// aiMarginFromLiveRange is the fallback used when the period has no snapshot
// row at all. It aggregates the WHOLE requested range: the previous version
// queried only the last day (`toDate`) and reported one day's spend as the
// period total, which is why a fresh database disagreed with
// `usage-by-operation` by the length of the window.
//
// Cost, request count and tokens come from the same `ai_usage_events` ledger
// the operation panel reads; revenue comes from the subscription reader (the
// same half-open [from, to) contract). Two small differences are accepted for a
// fallback whose only job is to avoid reporting zero: `GetAIErrorBreakdown`
// counts `released` events only while the snapshot path also counts
// `unknown_usage`/`expired_unverified`, and `GetAIUsageByOperation` does not
// expose thinking tokens.
func (uc *AdminDashboardUseCase) aiMarginFromLiveRange(ctx context.Context, fromDate, toDate, labelFrom, labelTo time.Time) (*dto.AIMarginAnalyticsRes, error) {
	subStats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	rows, err := uc.aiCostReader.GetAIUsageByOperation(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	requests, err := uc.aiCostReader.GetTotalAIRequestCount(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	errorRows, err := uc.aiCostReader.GetAIErrorBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	totalRev := 0.0
	if subStats != nil {
		totalRev = subStats.TotalSubscriptionRevenue.InexactFloat64()
	}

	var totalCostMicro int64
	var totalTokens int64
	for _, r := range rows {
		totalCostMicro += r.TotalCostMicroVND
		totalTokens += r.TotalPromptTokens + r.TotalOutputTokens
	}

	var totalErrors int64
	for _, r := range errorRows {
		totalErrors += r.ErrorCount
	}

	actualCostVND := float64(totalCostMicro) / microVNDDivisor
	netMargin := totalRev - actualCostVND

	marginPct := 0.0
	if totalRev > 0 {
		marginPct = (netMargin / totalRev) * 100
	}

	return &dto.AIMarginAnalyticsRes{
		FromDate:                labelFrom,
		ToDate:                  labelTo,
		TotalSubscriptionRevVND: totalRev,
		TotalAIActualCostVND:    actualCostVND,
		NetAIMarginVND:          netMargin,
		MarginPercentage:        marginPct,
		TotalAIPaidTokens:       totalTokens,
		AIRequestsCount:         requests,
		AIErrorCount:            totalErrors,
	}, nil
}

// GetPayOSReconciliation aggregates the transaction table over the selected
// period instead of reading the latest snapshot (029 R6/FR-017). The counters
// span both transaction types (direct purchase + wallet top-up), and
// `totalSuccessValue` is the real SUM(amount) of the successful ones — the
// panel no longer borrows the subscription-only revenue.
func (uc *AdminDashboardUseCase) GetPayOSReconciliation(ctx context.Context, from, to time.Time) (*dto.PayOSReconciliationRes, error) {
	if from.IsZero() && to.IsZero() {
		// No period sent: today by the VN calendar (data-model §3). A single
		// sent bound leaves the other side open.
		now := timeutils.NowVN()
		from, to = timeutils.VNDayRange(now)
	}

	stats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return &dto.PayOSReconciliationRes{
			TotalTransactions: 0,
			SuccessCount:      0,
			FailedCount:       0,
			PendingCount:      0,
			VoidedCount:       0,
			TotalSuccessValue: 0,
		}, nil
	}

	return &dto.PayOSReconciliationRes{
		TotalTransactions: stats.PayOSTransactionCount,
		SuccessCount:      stats.PayOSSuccessCount,
		FailedCount:       stats.PayOSFailedCount,
		PendingCount:      stats.PayOSPendingCount,
		VoidedCount:       stats.PayOSVoidedCount,
		TotalSuccessValue: stats.TotalPayOSSuccessValue.InexactFloat64(),
	}, nil
}

// GetSubscriptionDistribution forwards the period to the Account service and
// renders its interval-based result (031 D4/D5). The Account side already
// clipped a partial period to the cutover and decided the completeness; this
// use case only formats the cutover instant as a VN calendar date and keeps
// the percentage/totalUsers formulas on the bucket sum (FR-011).
func (uc *AdminDashboardUseCase) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) (*dto.SubscriptionDistributionRes, error) {
	result, err := uc.subscriptionReader.GetSubscriptionDistribution(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &dto.SubscriptionDistributionRes{
			Distribution: make([]dto.SubscriptionDistributionDTO, 0),
		}, nil
	}

	totalUsers := 0
	distribution := make([]dto.SubscriptionDistributionDTO, 0, len(result.Items))
	for _, r := range result.Items {
		totalUsers += int(r.UserCount)
		distribution = append(distribution, dto.SubscriptionDistributionDTO{
			PlanCode:   r.PlanCode,
			UserCount:  int(r.UserCount),
			Percentage: 0,
		})
	}

	for i := range distribution {
		if totalUsers > 0 {
			distribution[i].Percentage = float64(distribution[i].UserCount) / float64(totalUsers) * 100
		}
	}

	// The wire carries an instant; the response contract is the VN calendar
	// date. vnInclusiveDayLabel is the shared formatter used by the other
	// period responses, so the date cannot disagree with their day boundaries.
	var dataAvailableFrom *string
	if result.DataAvailableFrom != nil {
		label := vnInclusiveDayLabel(*result.DataAvailableFrom)
		dataAvailableFrom = &label
	}

	return &dto.SubscriptionDistributionRes{
		Distribution:      distribution,
		TotalUsers:        totalUsers,
		DataAvailableFrom: dataAvailableFrom,
		DataCompleteness:  result.DataCompleteness,
	}, nil
}

func (uc *AdminDashboardUseCase) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) (*dto.RevenueBreakdownRes, error) {
	// Same defaults as before (6 months back, ending now) — the intentional
	// FR-012 change is only that the edges are VN calendar-day boundaries.
	if fromDate.IsZero() {
		fromDate = timeutils.StartOfDayInVN(timeutils.NowVN().AddDate(0, -6, 0))
	}
	if toDate.IsZero() {
		toDate = timeutils.EndOfDayInVN(timeutils.NowVN())
	}

	rows, err := uc.subscriptionReader.GetRevenueBreakdown(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	totalRevenue := 0.0
	breakdown := make([]dto.RevenueBreakdownDTO, 0, len(rows))
	for _, r := range rows {
		revenue, _ := r.Revenue.Float64()
		totalRevenue += revenue
		breakdown = append(breakdown, dto.RevenueBreakdownDTO{
			PlanCode:         r.PlanCode,
			Month:            r.Month.Format("2006-01"),
			TransactionCount: int(r.TransactionCount),
			RevenueVND:       revenue,
		})
	}

	return &dto.RevenueBreakdownRes{
		FromDate:        vnInclusiveDayLabel(fromDate),
		ToDate:          vnInclusiveDayLabel(toDate.Add(-time.Nanosecond)),
		Breakdown:       breakdown,
		TotalRevenueVND: totalRevenue,
	}, nil
}

func (uc *AdminDashboardUseCase) GetAIUsageByOperation(ctx context.Context, fromDate, toDate time.Time) (*dto.AIUsageByOperationRes, error) {
	// The same 30-day defaults, now bounded by VN calendar days (029 R1).
	if fromDate.IsZero() {
		fromDate = timeutils.StartOfDayInVN(timeutils.NowVN().AddDate(0, 0, -30))
	}
	if toDate.IsZero() {
		toDate = timeutils.EndOfDayInVN(timeutils.NowVN())
	}

	rows, err := uc.aiCostReader.GetAIUsageByOperation(ctx, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	operations := make([]dto.AIUsageByOperationDTO, 0, len(rows))
	for _, r := range rows {
		operations = append(operations, dto.AIUsageByOperationDTO{
			Operation:         r.Operation,
			LogicalRoute:      r.LogicalRoute,
			RequestCount:      r.RequestCount,
			TotalCostVND:      float64(r.TotalCostMicroVND) / microVNDDivisor,
			TotalPromptTokens: r.TotalPromptTokens,
			TotalOutputTokens: r.TotalOutputTokens,
		})
	}

	return &dto.AIUsageByOperationRes{
		FromDate:   vnInclusiveDayLabel(fromDate),
		ToDate:     vnInclusiveDayLabel(toDate.Add(-time.Nanosecond)),
		Operations: operations,
	}, nil
}

// GetAIErrorBreakdown reports the AI error rate over ONE period (FR-016).
//
// The old implementation counted errors all-time while the request sample
// covered the last 30 days, which is how the rate could exceed 100%. Now the
// numerator (sum of the grouped breakdown), the denominator and the groups all
// share `[from, to)`; absent bounds default to the last 30 VN calendar days.
// A single sent bound leaves the other side open.
func (uc *AdminDashboardUseCase) GetAIErrorBreakdown(ctx context.Context, from, to time.Time) (*dto.AIErrorBreakdownRes, error) {
	if from.IsZero() && to.IsZero() {
		now := timeutils.NowVN()
		from = timeutils.StartOfDayInVN(now.AddDate(0, 0, -30))
		to = timeutils.EndOfDayInVN(now)
	}

	rows, err := uc.aiCostReader.GetAIErrorBreakdown(ctx, from, to)
	if err != nil {
		return nil, err
	}

	totalErrors := int64(0)
	breakdown := rows
	if breakdown == nil {
		breakdown = make([]aiexecutor_dto.AIErrorBreakdownDTO, 0)
	}
	for _, r := range rows {
		totalErrors += r.ErrorCount
	}

	totalRequests, _ := uc.aiCostReader.GetTotalAIRequestCount(ctx, from, to)

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

// GetRenewalStats returns renewal-shot statistics over the half-open period
// [from, to) (029 R8: attempts created in the period). Both bounds zero keeps
// the all-time behaviour; the account service owns the filtering.
//
// The success rate counts attempts that reached a decisive payment outcome
// (succeeded or failed) as its denominator: non-payment outcomes (skipped,
// downgraded, renewed) are legitimate renewals, not failures, so counting them
// would deflate the rate.
func (uc *AdminDashboardUseCase) GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsRes, error) {
	stats, err := uc.subscriptionReader.GetRenewalStats(ctx, from, to)
	if err != nil {
		return nil, err
	}

	decided := stats.SuccessCount + stats.FailedCount
	successRate := 0.0
	if decided > 0 {
		successRate = float64(stats.SuccessCount) / float64(decided) * 100
	}

	return &dto.RenewalStatsRes{
		TotalAttempts: int(stats.TotalAttempts),
		SuccessCount:  int(stats.SuccessCount),
		FailedCount:   int(stats.FailedCount),
		SuccessRate:   successRate,
		AvgRetryCount: stats.AvgRetryCount,
	}, nil
}

func (uc *AdminDashboardUseCase) GetSystemOverview(ctx context.Context) (*dto.AdminOverviewRes, error) {
	kpi, err := uc.GetAdminKPICards(ctx)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowVN()
	aiMargin, _ := uc.GetAIMarginAnalytics(ctx, now.AddDate(0, 0, -dashboard_app.DefaultLookbackDays), now)

	// Zero bounds: the payOS panel applies its today (VN) default (029 T035).
	payos, _ := uc.GetPayOSReconciliation(ctx, time.Time{}, time.Time{})

	return &dto.AdminOverviewRes{
		KPICards:       kpi,
		AIMargin:       aiMargin,
		PayOSReconcile: payos,
	}, nil
}
