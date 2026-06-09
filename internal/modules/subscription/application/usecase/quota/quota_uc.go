package quota

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type UserQuotaUseCase struct {
	quotaRepo    repositories.IUserDailyQuotaRepository
	subRepo      repositories.IUserSubscriptionRepository
	planRepo     repositories.ISubscriptionPlanRepository
	stateSupport *subscription.SubscriptionStateSupport
}

func NewUserQuotaUseCase(
	quotaRepo repositories.IUserDailyQuotaRepository,
	subRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
) uc_interfaces.IUserQuotaUseCase {
	return &UserQuotaUseCase{
		quotaRepo:    quotaRepo,
		subRepo:      subRepo,
		planRepo:     planRepo,
		stateSupport: subscription.NewSubscriptionStateSupport(subRepo, planRepo, quotaRepo),
	}
}

// checkAndResetDailyQuota fetches the user's daily quota and lazily performs reset if a new day has arrived
func (uc *UserQuotaUseCase) checkAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := uc.stateSupport.GetOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lastReset := quota.LastResetDate
	if now.Year() > lastReset.Year() ||
		(now.Year() == lastReset.Year() && now.Month() > lastReset.Month()) ||
		(now.Year() == lastReset.Year() && now.Month() == lastReset.Month() && now.Day() > lastReset.Day()) {

		quota.OutfitRecommendCount = 0
		quota.AiUsageCount = 0
		quota.LastResetDate = now

		err = uc.quotaRepo.Update(ctx, quota)
		if err != nil {
			return nil, err
		}
	}

	return quota, nil
}

// GetAndResetDailyQuota evaluates daily resets lazily and retrieves the fresh resource counters
func (uc *UserQuotaUseCase) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
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

	return subscription.BuildUserSubscriptionDTO(sub, plan, quota), nil
}

// UpdateOutfitQuota alters daily recommended outfit generations count
func (uc *UserQuotaUseCase) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
	if err != nil {
		return err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return err
	}

	newCount := quota.OutfitRecommendCount + count
	if newCount > plan.AiOutfitDailyQuota {
		return subscriptionerrors.ErrAiOutfitQuotaExceeded
	}

	quota.OutfitRecommendCount = newCount
	return uc.quotaRepo.Update(ctx, quota)
}

// UpdateAiChatQuota alters daily AI chatbot usage count
func (uc *UserQuotaUseCase) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
	if err != nil {
		return err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return err
	}

	newCount := quota.AiUsageCount + count
	if newCount > plan.AiChatDailyQuota {
		return subscriptionerrors.ErrAiChatQuotaExceeded
	}

	quota.AiUsageCount = newCount
	return uc.quotaRepo.Update(ctx, quota)
}
