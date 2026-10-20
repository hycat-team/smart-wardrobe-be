package admin_stats

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
)

// The three values `dataCompleteness` can carry (031 FR-012, contract
// distribution-api-delta §1). They cross the wire as the proto enum
// DistributionDataCompleteness and reach JSON as the lowercase string.
const (
	DistributionCompletenessComplete    = "complete"
	DistributionCompletenessPartial     = "partial"
	DistributionCompletenessUnavailable = "unavailable"
)

type AdminStatsUseCase struct {
	depositRepo    repositories.IDepositTransactionRepository
	renewalRepo    repositories.ISubscriptionRenewalAttemptRepository
	planPeriodRepo repositories.IUserPlanPeriodRepository
	log            logger.Interface
}

func NewAdminStatsUseCase(
	depositRepo repositories.IDepositTransactionRepository,
	renewalRepo repositories.ISubscriptionRenewalAttemptRepository,
	planPeriodRepo repositories.IUserPlanPeriodRepository,
	log logger.Interface,
) uc_interfaces.IAdminStatsUseCase {
	return &AdminStatsUseCase{
		depositRepo:    depositRepo,
		renewalRepo:    renewalRepo,
		planPeriodRepo: planPeriodRepo,
		log:            log,
	}
}

func (uc *AdminStatsUseCase) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	return uc.depositRepo.GetAdminSubscriptionStats(ctx, from, to)
}

// GetSubscriptionDistribution answers "who held which plan inside [from, to)"
// from the validity intervals instead of the overwritten current-plan
// projection (031 FR-007).
//
// Completeness is decided BEFORE the query (plan D4/data-model §6):
//   - no bounds: the live snapshot, always complete;
//   - a period ending at/before the cutover: unavailable, no query is run
//     because the report cannot answer it without inventing history;
//   - a period starting before the cutover: partial, and the query starts at
//     the cutover so the buckets describe exactly the part that is known;
//   - a period entirely at/after the cutover: complete;
//   - a period with no cutover row: unavailable (the date is unknown, so
//     dataAvailableFrom stays nil), never a silent zero.
func (uc *AdminStatsUseCase) GetSubscriptionDistribution(ctx context.Context, from, to time.Time) (*dto.SubscriptionDistributionResultDTO, error) {
	cutover, err := uc.planPeriodRepo.GetHistoryCutover(ctx)
	if err != nil {
		return nil, err
	}

	var availableFrom *time.Time
	if cutover != nil {
		startedAt := cutover.StartedAt
		availableFrom = &startedAt
	} else if (!from.IsZero() || !to.IsZero()) && uc.log != nil {
		uc.log.Warn("subscription distribution: plan_history_meta cutover row is missing; the requested period is reported as unavailable")
	}

	queryFrom, completeness := resolveDistributionWindow(from, to, availableFrom)
	result := &dto.SubscriptionDistributionResultDTO{
		Items:             []dto.SubscriptionDistributionDTO{},
		DataAvailableFrom: availableFrom,
		DataCompleteness:  completeness,
	}

	if completeness == DistributionCompletenessUnavailable {
		return result, nil
	}

	items, err := uc.planPeriodRepo.GetDistribution(ctx, queryFrom, to)
	if err != nil {
		return nil, err
	}
	if items != nil {
		result.Items = items
	}
	return result, nil
}

func (uc *AdminStatsUseCase) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	return uc.depositRepo.GetRevenueBreakdown(ctx, fromDate, toDate)
}

func (uc *AdminStatsUseCase) GetRenewalStats(ctx context.Context, from, to time.Time) (*dto.RenewalStatsDTO, error) {
	return uc.renewalRepo.GetRenewalStats(ctx, from, to)
}

// resolveDistributionWindow applies the completeness matrix (031 data-model
// §6). A zero bound is infinity: it must never be compared as the zero
// timestamp, which would put every period before the cutover or make the
// no-period snapshot look partial.
//
// The returned `from` is the query lower bound: it is raised to the cutover
// when the requested period starts before it, so a partial answer counts only
// the part it can actually prove.
func resolveDistributionWindow(from, to time.Time, cutover *time.Time) (time.Time, string) {
	if from.IsZero() && to.IsZero() {
		return from, DistributionCompletenessComplete
	}
	if cutover == nil {
		return from, DistributionCompletenessUnavailable
	}
	if !to.IsZero() && !to.After(*cutover) {
		return from, DistributionCompletenessUnavailable
	}
	if from.IsZero() || from.Before(*cutover) {
		return *cutover, DistributionCompletenessPartial
	}
	return from, DistributionCompletenessComplete
}
