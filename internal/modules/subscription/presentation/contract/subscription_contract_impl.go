package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptioncaching "smart-wardrobe-be/internal/modules/subscription/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/mapper"

	"github.com/google/uuid"
)

type UserSubscriptionContract struct {
	subUC         uc_interfaces.ISubscriptionUseCase
	overviewCache subscriptioncaching.ISubscriptionOverviewCache
}

func NewUserSubscriptionContract(subUC uc_interfaces.ISubscriptionUseCase, overviewCache subscriptioncaching.ISubscriptionOverviewCache) IUserSubscriptionContract {
	return &UserSubscriptionContract{subUC: subUC, overviewCache: overviewCache}
}

// GetUserSubscriptionOverview serves the AI hot path from the per-user cache
// (spec 019 FR-007): hit → contract DTO; miss/failure → use case + fill cache.
// Daily quota counters are never cached — they flow through IUserQuotaContract.
func (c *UserSubscriptionContract) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionContractDTO, error) {
	if entry, err := c.overviewCache.Get(ctx, userID); err == nil && entry != nil {
		return mapper.CacheEntryToContractDTO(entry), nil
	}

	res, err := c.subUC.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}
	contractDTO := mapper.MapUserSubscriptionOverviewToContractDTO(res)
	_ = c.overviewCache.Set(ctx, userID, mapper.ContractDTOToCacheEntry(contractDTO))
	return contractDTO, nil
}

func (c *UserSubscriptionContract) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionContractDTO, error) {
	resMap, err := c.subUC.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]*dto.UserSubscriptionContractDTO, len(resMap))
	for userID, res := range resMap {
		result[userID] = mapper.MapUserSubscriptionOverviewToContractDTO(res)
	}
	return result, nil
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
