package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"

	"github.com/google/uuid"
)

type UserSubscriptionContract struct {
	subUC uc_interfaces.ISubscriptionUseCase
}

func NewUserSubscriptionContract(subUC uc_interfaces.ISubscriptionUseCase) IUserSubscriptionContract {
	return &UserSubscriptionContract{subUC: subUC}
}

func (c *UserSubscriptionContract) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewDTO, error) {
	return c.subUC.GetUserSubscriptionOverview(ctx, userID)
}

func (c *UserSubscriptionContract) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewDTO, error) {
	return c.subUC.GetUserSubscriptionOverviews(ctx, userIDs)
}

type UserQuotaContract struct {
	quotaUC uc_interfaces.IUserQuotaUseCase
}

func NewUserQuotaContract(quotaUC uc_interfaces.IUserQuotaUseCase) IUserQuotaContract {
	return &UserQuotaContract{quotaUC: quotaUC}
}

func (c *UserQuotaContract) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error) {
	return c.quotaUC.GetAndResetDailyQuota(ctx, userID)
}

func (c *UserQuotaContract) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error {
	return c.quotaUC.UpdateOutfitQuota(ctx, userID, count)
}

func (c *UserQuotaContract) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	return c.quotaUC.UpdateAiChatQuota(ctx, userID, count)
}
