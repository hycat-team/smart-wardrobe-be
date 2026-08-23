package mapper

import (
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"

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

// BuildUserSubscriptionOverviewDTO constructs a contract UserSubscriptionOverviewDTO from the database structures.
func BuildUserSubscriptionOverviewDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *dto.UserSubscriptionOverviewDTO {
	if sub == nil || plan == nil {
		return nil
	}
	expiresAt, autoRenew := sub.ExpiresAt, sub.IsAutoRenewEnabled
	if plan.ID != sub.SubscriptionPlanID {
		expiresAt = nil
		autoRenew = false
	}
	return &dto.UserSubscriptionOverviewDTO{
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
