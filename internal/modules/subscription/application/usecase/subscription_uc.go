package usecase

import (
	"context"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"

	"github.com/google/uuid"
)

type SubscriptionUseCase struct {
	subscriptionContract contract.ISubscriptionModuleContract
}

func NewSubscriptionUseCase(subContract contract.ISubscriptionModuleContract) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		subscriptionContract: subContract,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	return uc.subscriptionContract.GetAndResetDailyQuota(ctx, userID)
}
