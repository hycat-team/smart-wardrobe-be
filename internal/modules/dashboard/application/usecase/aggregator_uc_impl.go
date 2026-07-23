package usecase

import (
	"context"
	"time"

	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	repo_interfaces "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"

	"go.uber.org/zap"
)

type DashboardAggregatorUseCase struct {
	logger             logger.Interface
	dashboardRepo      repo_interfaces.IDashboardRepository
	userReader         identity_contract.IUserDashboardReader
	brandReader        brand_contract.IBrandDashboardReader
	aiCostReader       aiexecutor_contract.IAICostDashboardReader
	subscriptionReader subscription_contract.ISubscriptionDashboardReader
}

func NewDashboardAggregatorUseCase(
	l logger.Interface,
	dashboardRepo repo_interfaces.IDashboardRepository,
	userReader identity_contract.IUserDashboardReader,
	brandReader brand_contract.IBrandDashboardReader,
	aiCostReader aiexecutor_contract.IAICostDashboardReader,
	subscriptionReader subscription_contract.ISubscriptionDashboardReader,
) uc_interfaces.IDashboardAggregatorUseCase {
	return &DashboardAggregatorUseCase{
		logger:             l,
		dashboardRepo:      dashboardRepo,
		userReader:         userReader,
		brandReader:        brandReader,
		aiCostReader:       aiCostReader,
		subscriptionReader: subscriptionReader,
	}
}

func (uc *DashboardAggregatorUseCase) AggregateAdminSnapshots(ctx context.Context) error {
	today := time.Now().Truncate(24 * time.Hour)

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	userStats, err := uc.userReader.GetAdminUserStats(ctxTimeout, today)
	if err != nil {
		uc.logger.Error("Failed to read admin user stats for snapshot", zap.Error(err))
		return err
	}

	brandStats, err := uc.brandReader.GetAdminBrandStats(ctxTimeout, today)
	if err != nil {
		uc.logger.Error("Failed to read admin brand stats for snapshot", zap.Error(err))
		return err
	}

	aiStats, err := uc.aiCostReader.GetAdminAIUsageStats(ctxTimeout, today)
	if err != nil {
		uc.logger.Error("Failed to read AI cost stats for snapshot", zap.Error(err))
		return err
	}

	subStats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctxTimeout, today)
	if err != nil {
		uc.logger.Error("Failed to read subscription stats for snapshot", zap.Error(err))
		return err
	}

	snapshot := &entities.DashboardAdminDailySnapshot{
		SnapshotDate: today,
	}

	if userStats != nil {
		snapshot.TotalUsers = userStats.TotalUsers
		snapshot.NewUsers = userStats.NewUsers
		snapshot.ActiveUsersDau = userStats.ActiveUsersDau
	}

	if brandStats != nil {
		snapshot.TotalBrands = brandStats.TotalBrands
		snapshot.PendingReviewBrands = brandStats.PendingReviewBrands
	}

	if aiStats != nil {
		snapshot.AIRequestsCount = int64(aiStats.AIRequestsCount)
		snapshot.AIErrorCount = int64(aiStats.AIErrorCount)
		snapshot.AIPaidTokens = aiStats.AIPaidTokens
		snapshot.AIActualCostMicroVND = aiStats.AIActualCostMicroVND
		snapshot.NetAIMarginVND = snapshot.SubscriptionRevenueVND - (float64(aiStats.AIActualCostMicroVND) / microVNDDivisor)
	}

	if subStats != nil {
		rev, _ := subStats.TotalSubscriptionRevenue.Float64()
		snapshot.SubscriptionRevenueVND = rev
		snapshot.PayOSTransactionCount = subStats.PayOSTransactionCount
		snapshot.PayOSSuccessCount = subStats.PayOSSuccessCount
		if aiStats != nil {
			snapshot.NetAIMarginVND = rev - (float64(aiStats.AIActualCostMicroVND) / microVNDDivisor)
		}
	}

	return uc.dashboardRepo.UpsertAdminSnapshot(ctx, snapshot)
}

func (uc *DashboardAggregatorUseCase) AggregateBrandSnapshots(ctx context.Context) error {
	today := time.Now().Truncate(24 * time.Hour)
	uc.logger.Info("Aggregating brand daily snapshots for date", zap.String("date", today.Format("2006-01-02")))

	brandIDs, err := uc.brandReader.GetActiveBrandIDs(ctx)
	if err != nil {
		uc.logger.Error("Failed to fetch active brand IDs for daily snapshot", zap.Error(err))
		return err
	}

	for _, brandID := range brandIDs {
		stats, err := uc.brandReader.GetBrandAnalyticsStats(ctx, brandID, today)
		if err != nil || stats == nil {
			continue
		}

		snapshot := &entities.DashboardBrandDailySnapshot{
			BrandID:                   brandID,
			SnapshotDate:              today,
			TotalMembers:              stats.TotalMembers,
			PointsIssued:              stats.PointsIssued,
			PointsRedeemed:            stats.PointsRedeemed,
			PointsExpiring30d:         stats.PointsExpiring30d,
			PointsLiabilityValueVND:   stats.PointsLiabilityValueVND,
			ActiveVouchers:            stats.ActiveVouchers,
			VouchersRedeemed:          stats.VouchersRedeemed,
			DigitalSampleVotesCount:   stats.DigitalSampleVotesCount,
			DigitalSampleAvgRating:    stats.DigitalSampleAvgRating,
			OpenCSTickets:             stats.OpenCSTickets,
			CSFirstResponseAvgSeconds: stats.CSFirstResponseAvgSeconds,
		}

		if err := uc.dashboardRepo.UpsertBrandSnapshot(ctx, snapshot); err != nil {
			uc.logger.Error("Failed to upsert brand daily snapshot", zap.String("brandID", brandID.String()), zap.Error(err))
		}
	}
	return nil
}
