package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
)

type SubscriptionPlanUseCase struct {
	planRepo repositories.ISubscriptionPlanRepository
}

func NewSubscriptionPlanUseCase(
	planRepo repositories.ISubscriptionPlanRepository,
) uc_interfaces.ISubscriptionPlanUseCase {
	return &SubscriptionPlanUseCase{
		planRepo: planRepo,
	}
}

// GetPlans retrieves all subscription plans
func (uc *SubscriptionPlanUseCase) GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtoPlans := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		dtoPlans = append(dtoPlans, &dto.SubscriptionPlanDTO{
			ID:                 plan.ID,
			Slug:               plan.Slug,
			Name:               plan.Name,
			Price:              sharedmoney.ToFloat(plan.Price),
			MaxWardrobeItems:   plan.MaxWardrobeItems,
			MaxOutfits:         plan.MaxOutfits,
			AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
			AiChatDailyQuota:   plan.AiChatDailyQuota,
			DurationDays:       plan.DurationDays,
		})
	}
	return dtoPlans, nil
}

// GetDefaultSubscriptionPlanID retrieves default free tier ID
func (uc *SubscriptionPlanUseCase) GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error) {
	plan, err := uc.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if plan == nil {
		return uuid.Nil, subscriptionerrors.ErrDefaultPlanConfigNotFound
	}
	return plan.ID, nil
}

// IsPremiumPlan checks if plan corresponds to a premium price tier
func (uc *SubscriptionPlanUseCase) IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error) {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		return false, err
	}
	if plan == nil || !plan.IsActive {
		return false, nil
	}
	return plan.Price.GreaterThan(sharedmoney.Zero), nil
}

