package quota

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type UserQuotaUseCase struct {
	quotaRepo    repositories.IUserDailyQuotaRepository
	subRepo      repositories.IUserSubscriptionRepository
	planRepo     repositories.ISubscriptionPlanRepository
	stateSupport *subscription.SubscriptionStateSupport
	// 025 T063: the unit of work, held here so the increment paths can bracket
	// their read-modify-write in a transaction of their own.
	//
	// It was already a constructor argument — `SubscriptionStateSupport` needs it to
	// provision the default subscription — so nothing about the wiring changes; it
	// is kept as a field because the paths that need it are in THIS type.
	uow shared_repos.IUnitOfWork
}

func NewUserQuotaUseCase(
	quotaRepo repositories.IUserDailyQuotaRepository,
	subRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	planChangePublisher *subscription.PlanChangePublisher,
	// 024 T126: the state support opens a transaction to provision the default
	// subscription, so it needs the unit of work.
	// 025 T063: and so does the increment path below, for a different reason.
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IUserQuotaUseCase {
	return &UserQuotaUseCase{
		quotaRepo:    quotaRepo,
		subRepo:      subRepo,
		planRepo:     planRepo,
		stateSupport: subscription.NewSubscriptionStateSupport(subRepo, planRepo, quotaRepo, planChangePublisher, uow),
		uow:          uow,
	}
}

// checkAndResetDailyQuota fetches the user's daily quota and lazily performs reset if a new day has arrived.
//
// 025 T066: the read stays LOCK-FREE and the write opens a transaction only when
// there is genuinely a new day to record.
//
// Two rules pull in opposite directions here and both are requirements. FR-055
// says a write must not block a read for longer than it has to — and this read
// runs on EVERY AI request, so making it queue behind the increment path would
// serialise the AI path behind its own accounting. FR-054 says the counter must
// return to zero on a new day with nothing more than a request, which needs a
// write.
//
// The resolution is to decide BEFORE taking any lock. The common case — same day
// — costs one unlocked read and one comparison, exactly as before. Only the once-a-
// day rollover pays for a transaction, and even then the row is re-read under
// `FOR UPDATE` first, because two requests crossing midnight would otherwise both
// decide to reset and both write.
func (uc *UserQuotaUseCase) checkAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := uc.stateSupport.GetOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !isNewDay(quota.LastResetDate) {
		return quota, nil
	}

	return uc.resetQuotaForNewDay(ctx, userID)
}

// resetQuotaForNewDay zeroes the counters inside a transaction, re-reading the row
// under `FOR UPDATE` first.
//
// The re-read is not belt-and-braces. Two requests crossing midnight both see
// yesterday's date, both open a transaction, and without the lock the second
// writes a value derived from a read the first has already superseded. With it, the
// second observes the first's `LastResetDate`, finds the day has not changed, and
// does nothing.
func (uc *UserQuotaUseCase) resetQuotaForNewDay(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	var reset *entities.UserDailyQuota
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		quota, err := uc.quotaRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if quota == nil {
			// `GetOrCreateUserDailyQuota` above proved the row exists.
			return subscriptionerrors.ErrUserDailyQuotaNotFound()
		}
		if !isNewDay(quota.LastResetDate) {
			// Another request already reset it while this one waited.
			reset = quota
			return nil
		}
		quota.OutfitRecommendCount = 0
		quota.AiUsageCount = 0
		quota.LastResetDate = time.Now()
		if err := uc.quotaRepo.Update(txCtx, quota); err != nil {
			return err
		}
		reset = quota
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reset, nil
}

// isNewDay compares calendar days, not elapsed hours.
//
// A 24-hour window would reset the counter at 07:00 in the morning for an account
// that last used it at 23:00 — handing out a second full allowance on the same
// working day. The customer's day boundary is the calendar's, and the zone is the
// service's configured one (R6, constitution §VIII).
func isNewDay(lastReset time.Time) bool {
	now := time.Now()
	return now.Year() > lastReset.Year() ||
		(now.Year() == lastReset.Year() && now.Month() > lastReset.Month()) ||
		(now.Year() == lastReset.Year() && now.Month() == lastReset.Month() && now.Day() > lastReset.Day())
}

// GetAndResetDailyQuota evaluates daily resets lazily and retrieves the fresh resource counters
func (uc *UserQuotaUseCase) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error) {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return mapper.BuildUserSubscriptionDTO(sub, plan, quota), nil
}

// UpdateOutfitQuota alters daily recommended outfit generations count
//
// 025 T064: the read → add → write is now ONE critical section, taken under
// `FOR UPDATE`.
//
// Before this it was five statements with no transaction. Two concurrent AI calls
// both read 29, both computed 30, and both wrote 30 — the customer was charged for
// two generations and got one, and the counter drifted a little further out of
// step with reality on every request until the daily reset hid it. Nothing errored
// at any point.
//
// The two reads that stay OUTSIDE the transaction are `GetOrCreateUserSubscription`
// and `LoadPlanForSubscription`, and keeping them out is a requirement rather than
// an oversight: both open a transaction of their own, and `GormUnitOfWork.Execute`
// runs on the BASE handle (`shared/infrastructure/db/uow.go:69`), so calling them
// from inside would open a SECOND transaction on a SECOND connection that cannot
// see the outer one's rows — and the plan's quota would then be compared against a
// subscription the inner transaction created (R9).
func (uc *UserQuotaUseCase) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return err
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		quota, err := uc.quotaRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if quota == nil {
			// `GetOrCreateUserDailyQuota` above — reached through
			// `checkAndResetDailyQuota` — proved the row exists. Returning a nil
			// quota here would reset the customer's counters to zero.
			return subscriptionerrors.ErrUserDailyQuotaNotFound()
		}
		if isNewDay(quota.LastResetDate) {
			// The rollover is applied INSIDE the same critical section as the
			// increment, or a request that lands just after midnight would count
			// yesterday's usage against today's fresh allowance.
			quota.OutfitRecommendCount = 0
			quota.AiUsageCount = 0
			quota.LastResetDate = time.Now()
		}

		newCount := quota.OutfitRecommendCount + count
		if newCount > plan.AiOutfitDailyQuota {
			return subscriptionerrors.ErrAiOutfitQuotaExceeded()
		}

		quota.OutfitRecommendCount = newCount
		return uc.quotaRepo.Update(txCtx, quota)
	})
}

// UpdateAiChatQuota alters daily AI chatbot usage count
//
// 025 T065: identical to `UpdateOutfitQuota`, for the chatbot counter. It had the
// same defect and, being a separate method, would have kept it after the first was
// fixed — which is why both are fixed here rather than by a shared helper: the two
// differ in which field they move and which limit they check, and merging them
// would have meant a boolean parameter deciding what a quota row means.
func (uc *UserQuotaUseCase) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return err
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		quota, err := uc.quotaRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if quota == nil {
			return subscriptionerrors.ErrUserDailyQuotaNotFound()
		}
		if isNewDay(quota.LastResetDate) {
			quota.OutfitRecommendCount = 0
			quota.AiUsageCount = 0
			quota.LastResetDate = time.Now()
		}

		newCount := quota.AiUsageCount + count
		if newCount > plan.AiChatDailyQuota {
			return subscriptionerrors.ErrAiChatQuotaExceeded()
		}

		quota.AiUsageCount = newCount
		return uc.quotaRepo.Update(txCtx, quota)
	})
}
