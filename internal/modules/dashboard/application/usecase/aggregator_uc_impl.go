package usecase

import (
	"context"
	"time"

	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	dashboarderrors "smart-wardrobe-be/internal/modules/dashboard/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/dashboard/domain/entities"
	repo_interfaces "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

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

func (uc *DashboardAggregatorUseCase) AggregateAdminSnapshots(ctx context.Context, run *workerlog.Run) error {
	today := time.Now().Truncate(24 * time.Hour)

	ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	userStats, err := uc.userReader.GetAdminUserStats(ctxTimeout, today)
	if err != nil {
		run.ChildError(uc.logger, "Failed to read admin user stats for snapshot", zap.Error(err))
		return err
	}
	if userStats == nil {
		err := dashboarderrors.ErrDashboardDataNotFound()
		run.ChildError(uc.logger, "Admin user stats returned nil for snapshot", zap.Error(err))
		return err
	}

	brandStats, err := uc.brandReader.GetAdminBrandStats(ctxTimeout, today)
	if err != nil {
		run.ChildError(uc.logger, "Failed to read admin brand stats for snapshot", zap.Error(err))
		return err
	}
	if brandStats == nil {
		err := dashboarderrors.ErrDashboardDataNotFound()
		run.ChildError(uc.logger, "Admin brand stats returned nil for snapshot", zap.Error(err))
		return err
	}

	aiStats, err := uc.aiCostReader.GetAdminAIUsageStats(ctxTimeout, today)
	if err != nil {
		run.ChildError(uc.logger, "Failed to read AI cost stats for snapshot", zap.Error(err))
		return err
	}
	if aiStats == nil {
		err := dashboarderrors.ErrDashboardDataNotFound()
		run.ChildError(uc.logger, "AI cost stats returned nil for snapshot", zap.Error(err))
		return err
	}

	subStats, err := uc.subscriptionReader.GetAdminSubscriptionStats(ctxTimeout, today)
	if err != nil {
		run.ChildError(uc.logger, "Failed to read subscription stats for snapshot", zap.Error(err))
		return err
	}
	if subStats == nil {
		err := dashboarderrors.ErrDashboardDataNotFound()
		run.ChildError(uc.logger, "Subscription stats returned nil for snapshot", zap.Error(err))
		return err
	}

	snapshot := &entities.DashboardAdminDailySnapshot{
		SnapshotDate:         today,
		TotalUsers:           userStats.TotalUsers,
		NewUsers:             userStats.NewUsers,
		ActiveUsersDau:       userStats.ActiveUsersDau,
		TotalBrands:          brandStats.TotalBrands,
		PendingReviewBrands:  brandStats.PendingReviewBrands,
		AIRequestsCount:      int64(aiStats.AIRequestsCount),
		AIErrorCount:         int64(aiStats.AIErrorCount),
		AIPaidTokens:         aiStats.AIPaidTokens,
		AIActualCostMicroVND: aiStats.AIActualCostMicroVND,
	}

	rev, _ := subStats.TotalSubscriptionRevenue.Float64()
	snapshot.SubscriptionRevenueVND = rev
	snapshot.PayOSTransactionCount = subStats.PayOSTransactionCount
	snapshot.PayOSSuccessCount = subStats.PayOSSuccessCount
	snapshot.NetAIMarginVND = rev - (float64(aiStats.AIActualCostMicroVND) / microVNDDivisor)

	run.AddTotal(1)
	if err := uc.dashboardRepo.UpsertAdminSnapshot(ctx, snapshot); err != nil {
		run.ChildError(uc.logger, "Failed to upsert admin daily snapshot", zap.Error(err))
		return err
	}
	run.AddSuccess(1)
	return nil
}

func (uc *DashboardAggregatorUseCase) AggregateBrandSnapshots(ctx context.Context, run *workerlog.Run) error {
	today := time.Now().Truncate(24 * time.Hour)

	brandIDs, err := uc.brandReader.GetActiveBrandIDs(ctx)
	if err != nil {
		run.ChildError(uc.logger, "Failed to fetch active brand IDs for daily snapshot", zap.Error(err))
		return err
	}

	for _, brandID := range brandIDs {
		stats, err := uc.brandReader.GetBrandAnalyticsStats(ctx, brandID, today)
		if err != nil || stats == nil {
			continue
		}

		snapshot := &entities.DashboardBrandDailySnapshot{
			BrandID:                 brandID,
			SnapshotDate:            today,
			TotalMembers:            stats.TotalMembers,
			PointsIssued:            stats.PointsIssued,
			PointsRedeemed:          stats.PointsRedeemed,
			PointsExpiring30d:       stats.PointsExpiring30d,
			PointsLiabilityValueVND: stats.PointsLiabilityValueVND,
			ActiveVouchers:          stats.ActiveVouchers,
			VouchersRedeemed:        stats.VouchersRedeemed,
			DigitalSampleVotesCount: stats.DigitalSampleVotesCount,
			DigitalSampleAvgRating:  stats.DigitalSampleAvgRating,
			OpenCSTickets:           stats.OpenCSTickets,
		}

		run.AddTotal(1)
		if err := uc.dashboardRepo.UpsertBrandSnapshot(ctx, snapshot); err != nil {
			run.ChildError(uc.logger, "Failed to upsert brand daily snapshot", zap.String("brandID", brandID.String()), zap.Error(err))
			continue
		}
		run.AddSuccess(1)
	}
	return nil
}
