package contract

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/billing/contract"
	"smart-wardrobe-be/internal/modules/billing/domain/repositories"

	"github.com/google/uuid"
)

type BillingModuleContractImpl struct {
	planRepo repositories.ISubscriptionPlanRepository
}

func NewBillingModuleContractImpl(planRepo repositories.ISubscriptionPlanRepository) contract.IBillingModuleContract {
	return &BillingModuleContractImpl{
		planRepo: planRepo,
	}
}

func (impl *BillingModuleContractImpl) GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error) {
	plan, err := impl.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if plan == nil {
		return uuid.Nil, errors.New("không tìm thấy gói cước Free mặc định trong cơ sở dữ liệu")
	}
	return plan.ID, nil
}

func (impl *BillingModuleContractImpl) IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error) {
	plan, err := impl.planRepo.FindByID(ctx, planID)
	if err != nil {
		return false, err
	}
	if plan == nil || !plan.IsActive {
		return false, nil
	}
	return plan.Price > 0, nil
}
