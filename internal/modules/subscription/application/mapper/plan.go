package mapper

import (
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
)

// MapToSubscriptionPlanDTO maps a SubscriptionPlan entity into a SubscriptionPlanDTO.
func MapToSubscriptionPlanDTO(plan *entities.SubscriptionPlan) *dto.SubscriptionPlanDTO {
	if plan == nil {
		return nil
	}
	return &dto.SubscriptionPlanDTO{
		ID:                 plan.ID,
		Slug:               plan.Slug,
		Name:               plan.Name,
		Price:              sharedmoney.ToFloat(plan.Price),
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
		DurationDays:       plan.DurationDays,
		PlanKind:           plan.PlanKind,
		TierRank:           plan.TierRank,
		IsActive:           plan.IsActive,
	}
}

// MapToSubscriptionPlanDTOList maps a slice of SubscriptionPlan entities into a slice of SubscriptionPlanDTOs.
func MapToSubscriptionPlanDTOList(plans []*entities.SubscriptionPlan) []*dto.SubscriptionPlanDTO {
	result := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		if plan != nil {
			result = append(result, MapToSubscriptionPlanDTO(plan))
		}
	}
	return result
}

func MapToSubscriptionPlanBrief(plan *entities.SubscriptionPlan) *dto.SubscriptionPlanBriefRes {
	if plan == nil {
		return nil
	}
	return &dto.SubscriptionPlanBriefRes{
		Slug:               plan.Slug,
		Name:               plan.Name,
		Price:              sharedmoney.ToFloat(plan.Price),
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
		DurationDays:       plan.DurationDays,
		PlanKind:           plan.PlanKind,
		TierRank:           plan.TierRank,
	}
}

func MapToSubscriptionPlanBriefList(plans []*entities.SubscriptionPlan) []*dto.SubscriptionPlanBriefRes {
	result := make([]*dto.SubscriptionPlanBriefRes, 0, len(plans))
	for _, plan := range plans {
		if plan != nil {
			result = append(result, MapToSubscriptionPlanBrief(plan))
		}
	}
	return result
}
