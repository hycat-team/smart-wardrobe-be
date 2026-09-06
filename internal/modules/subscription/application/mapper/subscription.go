package mapper

import (
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	caching "smart-wardrobe-be/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"time"

	"github.com/google/uuid"
)

// BuildUserSubscriptionDTO constructs a contract UserSubscriptionDTO from the database structures.
func BuildUserSubscriptionDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan, quota *entities.UserDailyQuota) *dto.UserSubscriptionDTO {
	if sub == nil || plan == nil || quota == nil {
		return nil
	}
	expiresAt, autoRenew := sub.ExpiresAt, sub.IsAutoRenewEnabled
	if plan.ID != sub.SubscriptionPlanID {
		expiresAt = nil
		autoRenew = false
	}
	return &dto.UserSubscriptionDTO{
		PlanID:               plan.ID,
		PlanName:             plan.Name,
		PlanSlug:             plan.Slug,
		ExpiresAt:            expiresAt,
		IsAutoRenewEnabled:   autoRenew,
		AiOutfitDailyQuota:   plan.AiOutfitDailyQuota,
		AiChatDailyQuota:     plan.AiChatDailyQuota,
		OutfitRecommendCount: quota.OutfitRecommendCount,
		AiUsageCount:         quota.AiUsageCount,
		LastResetDate:        quota.LastResetDate,
	}
}

// BuildUserSubscriptionOverviewRes constructs the HTTP API response overview from the database structures.
func BuildUserSubscriptionOverviewRes(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *dto.UserSubscriptionOverviewRes {
	if sub == nil || plan == nil {
		return nil
	}
	expiresAt, autoRenew := sub.ExpiresAt, sub.IsAutoRenewEnabled
	if plan.ID != sub.SubscriptionPlanID {
		expiresAt = nil
		autoRenew = false
	}
	return &dto.UserSubscriptionOverviewRes{
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanSlug:           plan.Slug,
		ExpiresAt:          expiresAt,
		IsAutoRenewEnabled: autoRenew,
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
	}
}

// MapUserSubscriptionOverviewToContractDTO converts the API response overview
// into the pure cross-module contract payload served by IUserSubscriptionContract.
func MapUserSubscriptionOverviewToContractDTO(res *dto.UserSubscriptionOverviewRes) *dto.UserSubscriptionContractDTO {
	if res == nil {
		return nil
	}
	return &dto.UserSubscriptionContractDTO{
		PlanID:             res.PlanID,
		PlanName:           res.PlanName,
		PlanSlug:           res.PlanSlug,
		ExpiresAt:          res.ExpiresAt,
		IsAutoRenewEnabled: res.IsAutoRenewEnabled,
		MaxWardrobeItems:   res.MaxWardrobeItems,
		MaxOutfits:         res.MaxOutfits,
		AiOutfitDailyQuota: res.AiOutfitDailyQuota,
		AiChatDailyQuota:   res.AiChatDailyQuota,
	}
}

// BuildAdminUserSubscriptionItemDTO constructs an AdminUserSubscriptionItemDTO.
func BuildAdminUserSubscriptionItemDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *dto.AdminUserSubscriptionItemDTO {
	if sub == nil {
		return nil
	}
	planName := sub.CurrentPlanCode
	planID := sub.SubscriptionPlanID
	if plan != nil {
		planName = plan.Name
		planID = plan.ID
	}
	return &dto.AdminUserSubscriptionItemDTO{
		UserID:             sub.UserID,
		PlanID:             planID,
		PlanName:           planName,
		PlanSlug:           sub.CurrentPlanCode,
		PlanKind:           sub.CurrentPlanKind,
		TierRank:           sub.CurrentTierRank,
		StartedAt:          sub.StartedAt,
		ExpiresAt:          sub.ExpiresAt,
		IsAutoRenewEnabled: sub.IsAutoRenewEnabled,
		CreatedAt:          sub.CreatedAt,
		UpdatedAt:          sub.UpdatedAt,
	}
}

// BuildAdminUserSubscriptionItemDTOList maps a list of user subscriptions.
func BuildAdminUserSubscriptionItemDTOList(subs []*entities.UserSubscription, planByID map[uuid.UUID]*entities.SubscriptionPlan) []*dto.AdminUserSubscriptionItemDTO {
	result := make([]*dto.AdminUserSubscriptionItemDTO, 0, len(subs))
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		var plan *entities.SubscriptionPlan
		if planByID != nil {
			plan = planByID[sub.SubscriptionPlanID]
		}
		if plan == nil && sub.SubscriptionPlan != nil {
			plan = sub.SubscriptionPlan
		}
		result = append(result, BuildAdminUserSubscriptionItemDTO(sub, plan))
	}
	return result
}

// ContractDTOToCacheEntry converts the contract payload into the cache entry.
func ContractDTOToCacheEntry(dto *dto.UserSubscriptionContractDTO) *caching.SubscriptionOverviewCacheEntry {
	if dto == nil {
		return nil
	}
	return &caching.SubscriptionOverviewCacheEntry{
		PlanID:             dto.PlanID,
		PlanName:           dto.PlanName,
		PlanSlug:           dto.PlanSlug,
		ExpiresAt:          dto.ExpiresAt,
		IsAutoRenewEnabled: dto.IsAutoRenewEnabled,
		MaxWardrobeItems:   dto.MaxWardrobeItems,
		MaxOutfits:         dto.MaxOutfits,
		AiOutfitDailyQuota: dto.AiOutfitDailyQuota,
		AiChatDailyQuota:   dto.AiChatDailyQuota,
		CachedAt:           time.Now().UTC()}
}

// CacheEntryToContractDTO converts a cached overview entry back into the
// contract payload served to AI hot paths.
func CacheEntryToContractDTO(entry *caching.SubscriptionOverviewCacheEntry) *dto.UserSubscriptionContractDTO {
	if entry == nil {
		return nil
	}
	return &dto.UserSubscriptionContractDTO{
		PlanID:             entry.PlanID,
		PlanName:           entry.PlanName,
		PlanSlug:           entry.PlanSlug,
		ExpiresAt:          entry.ExpiresAt,
		IsAutoRenewEnabled: entry.IsAutoRenewEnabled,
		MaxWardrobeItems:   entry.MaxWardrobeItems,
		MaxOutfits:         entry.MaxOutfits,
		AiOutfitDailyQuota: entry.AiOutfitDailyQuota,
		AiChatDailyQuota:   entry.AiChatDailyQuota,
	}
}
