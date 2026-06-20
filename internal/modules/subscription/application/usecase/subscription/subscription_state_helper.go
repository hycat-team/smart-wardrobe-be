package subscription

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type SubscriptionStateSupport struct {
	userSubRepo repositories.IUserSubscriptionRepository
	planRepo    repositories.ISubscriptionPlanRepository
	quotaRepo   repositories.IUserDailyQuotaRepository
}

func NewSubscriptionStateSupport(
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
) *SubscriptionStateSupport {
	return &SubscriptionStateSupport{
		userSubRepo: userSubRepo,
		planRepo:    planRepo,
		quotaRepo:   quotaRepo,
	}
}

func (s *SubscriptionStateSupport) GetOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	sub, err := s.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		defaultPlan, err := s.planRepo.GetDefaultPlan(ctx)
		if err != nil {
			return nil, err
		}
		if defaultPlan == nil {
			return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound()
		}

		sub = &entities.UserSubscription{
			UserID:                 userID,
			SubscriptionPlanID:     defaultPlan.ID,
			CurrentPlanCode:        defaultPlan.Slug,
			CurrentTierRank:        defaultPlan.TierRank,
			CurrentPlanKind:        plankind.DefaultFree,
			CurrentBenefitSnapshot: entities.JSONDocument(`{}`),
			StartedAt:              time.Now(),
			SubscriptionPlan:       defaultPlan,
		}
		if err := s.userSubRepo.ProvisionDefault(ctx, sub); err != nil {
			return nil, err
		}
		sub, err = s.userSubRepo.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (s *SubscriptionStateSupport) GetOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := s.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quota == nil {
		quota = &entities.UserDailyQuota{
			UserID:               userID,
			OutfitRecommendCount: 0,
			AiUsageCount:         0,
			LastResetDate:        time.Now(),
		}
		if err := s.quotaRepo.Create(ctx, quota); err != nil {
			return nil, err
		}
	}
	return quota, nil
}

func (s *SubscriptionStateSupport) LoadPlanForSubscription(ctx context.Context, sub *entities.UserSubscription) (*entities.SubscriptionPlan, error) {
	if sub.CurrentPlanKind == plankind.Finite && sub.ExpiresAt != nil && !sub.ExpiresAt.After(time.Now()) {
		if sub.FallbackPlanID != nil {
			plan, err := s.planRepo.GetByID(ctx, *sub.FallbackPlanID)
			if err != nil {
				return nil, err
			}
			if plan != nil {
				return plan, nil
			}
		}
		plan, err := s.planRepo.GetDefaultPlan(ctx)
		if err != nil {
			return nil, err
		}
		if plan == nil {
			return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound()
		}
		return plan, nil
	}
	if sub.SubscriptionPlan != nil {
		return sub.SubscriptionPlan, nil
	}

	plan, err := s.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrSubscriptionPlanNotFound()
	}
	return plan, nil
}

func (s *SubscriptionStateSupport) ResolveSubscriptionPlans(ctx context.Context, subs []*entities.UserSubscription) (map[uuid.UUID]*entities.SubscriptionPlan, error) {
	planByID := make(map[uuid.UUID]*entities.SubscriptionPlan, len(subs))
	missingPlanIDs := make([]uuid.UUID, 0)
	seenMissingPlanIDs := make(map[uuid.UUID]struct{})

	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if sub.SubscriptionPlan != nil {
			planByID[sub.SubscriptionPlanID] = sub.SubscriptionPlan
		} else if _, exists := seenMissingPlanIDs[sub.SubscriptionPlanID]; !exists {
			seenMissingPlanIDs[sub.SubscriptionPlanID] = struct{}{}
			missingPlanIDs = append(missingPlanIDs, sub.SubscriptionPlanID)
		}
		if sub.FallbackPlanID != nil {
			if _, exists := seenMissingPlanIDs[*sub.FallbackPlanID]; !exists {
				seenMissingPlanIDs[*sub.FallbackPlanID] = struct{}{}
				missingPlanIDs = append(missingPlanIDs, *sub.FallbackPlanID)
			}
		}
	}

	if len(missingPlanIDs) == 0 {
		return planByID, nil
	}

	plans, err := s.planRepo.GetByIDs(ctx, missingPlanIDs)
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		planByID[plan.ID] = plan
	}

	return planByID, nil
}

func BuildUserSubscriptionDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan, quota *entities.UserDailyQuota) *contract.UserSubscriptionDTO {
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

func BuildUserSubscriptionOverviewDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *contract.UserSubscriptionOverviewDTO {
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
