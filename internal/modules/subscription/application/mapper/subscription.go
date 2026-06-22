package mapper

import (
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// BuildUserSubscriptionDTO constructs a contract UserSubscriptionDTO from the database structures.
func BuildUserSubscriptionDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan, quota *entities.UserDailyQuota) *contract.UserSubscriptionDTO {
	if sub == nil || plan == nil || quota == nil {
		return nil
	}
	expiresAt, autoRenew := sub.ExpiresAt, sub.IsAutoRenewEnabled
	if plan.ID != sub.SubscriptionPlanID {
		expiresAt = nil
		autoRenew = false
	}
	return &contract.UserSubscriptionDTO{
		PlanID:               plan.ID,
		PlanName:             plan.Name,
		PlanSlug:             plan.Slug,
		PlanKind:             plan.PlanKind,
		TierRank:             plan.TierRank,
		ExpiresAt:            expiresAt,
		IsAutoRenewEnabled:   autoRenew,
		FallbackPlanCode:     sub.FallbackPlanCode,
		FallbackTierRank:     sub.FallbackTierRank,
		FallbackPlanKind:     sub.FallbackPlanKind,
		MaxWardrobeItems:     plan.MaxWardrobeItems,
		MaxOutfits:           plan.MaxOutfits,
		AiOutfitDailyQuota:   plan.AiOutfitDailyQuota,
		AiChatDailyQuota:     plan.AiChatDailyQuota,
		OutfitRecommendCount: quota.OutfitRecommendCount,
		AiUsageCount:         quota.AiUsageCount,
		LastResetDate:        quota.LastResetDate,
	}
}

// BuildUserSubscriptionOverviewDTO constructs a contract UserSubscriptionOverviewDTO from the database structures.
func BuildUserSubscriptionOverviewDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *contract.UserSubscriptionOverviewDTO {
	if sub == nil || plan == nil {
		return nil
	}
	expiresAt, autoRenew := sub.ExpiresAt, sub.IsAutoRenewEnabled
	if plan.ID != sub.SubscriptionPlanID {
		expiresAt = nil
		autoRenew = false
	}
	return &contract.UserSubscriptionOverviewDTO{
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanSlug:           plan.Slug,
		PlanKind:           plan.PlanKind,
		TierRank:           plan.TierRank,
		ExpiresAt:          expiresAt,
		IsAutoRenewEnabled: autoRenew,
		FallbackPlanCode:   sub.FallbackPlanCode,
		FallbackTierRank:   sub.FallbackTierRank,
		FallbackPlanKind:   sub.FallbackPlanKind,
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
	}
}
