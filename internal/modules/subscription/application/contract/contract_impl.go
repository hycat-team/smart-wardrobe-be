package contract

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
)

type SubscriptionModuleContractImpl struct {
	planRepo repositories.ISubscriptionPlanRepository
}

func NewSubscriptionModuleContractImpl(planRepo repositories.ISubscriptionPlanRepository) contract.ISubscriptionModuleContract {
	return &SubscriptionModuleContractImpl{
		planRepo: planRepo,
	}
}

func (impl *SubscriptionModuleContractImpl) GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error) {
	plan, err := impl.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if plan == nil {
		return uuid.Nil, errors.New("không tìm thấy gói cước Free mặc định trong cơ sở dữ liệu")
	}
	return plan.ID, nil
}

func (impl *SubscriptionModuleContractImpl) IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error) {
	plan, err := impl.planRepo.FindByID(ctx, planID)
	if err != nil {
		return false, err
	}
	if plan == nil || !plan.IsActive {
		return false, nil
	}
	return plan.Price > 0, nil
}
