package usecase

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type subscriptionStateSupport struct {
	userSubRepo repositories.IUserSubscriptionRepository
	planRepo    repositories.ISubscriptionPlanRepository
	quotaRepo   repositories.IUserDailyQuotaRepository
}

func newSubscriptionStateSupport(
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
) *subscriptionStateSupport {
	return &subscriptionStateSupport{
		userSubRepo: userSubRepo,
		planRepo:    planRepo,
		quotaRepo:   quotaRepo,
	}
}

func (s *subscriptionStateSupport) getOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
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
			return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound
		}

		sub = &entities.UserSubscription{
			UserID:             userID,
			SubscriptionPlanID: defaultPlan.ID,
			IsActive:           true,
			SubscriptionPlan:   defaultPlan,
		}
		if err := s.userSubRepo.Create(ctx, sub); err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (s *subscriptionStateSupport) getOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
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

func (s *subscriptionStateSupport) loadPlanForSubscription(ctx context.Context, sub *entities.UserSubscription) (*entities.SubscriptionPlan, error) {
	if sub.SubscriptionPlan != nil {
		return sub.SubscriptionPlan, nil
	}

	plan, err := s.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrSubscriptionPlanNotFound
	}
	return plan, nil
}

func (s *subscriptionStateSupport) resolveSubscriptionPlans(ctx context.Context, subs []*entities.UserSubscription) (map[uuid.UUID]*entities.SubscriptionPlan, error) {
	planByID := make(map[uuid.UUID]*entities.SubscriptionPlan, len(subs))
	missingPlanIDs := make([]uuid.UUID, 0)
	seenMissingPlanIDs := make(map[uuid.UUID]struct{})

	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if sub.SubscriptionPlan != nil {
			planByID[sub.SubscriptionPlanID] = sub.SubscriptionPlan
			continue
		}
		if _, exists := seenMissingPlanIDs[sub.SubscriptionPlanID]; exists {
			continue
		}
		seenMissingPlanIDs[sub.SubscriptionPlanID] = struct{}{}
		missingPlanIDs = append(missingPlanIDs, sub.SubscriptionPlanID)
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

func buildUserSubscriptionDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan, quota *entities.UserDailyQuota) *contract.UserSubscriptionDTO {
	return &contract.UserSubscriptionDTO{
		PlanID:               plan.ID,
		PlanName:             plan.Name,
		PlanSlug:             plan.Slug,
		ExpiresAt:            sub.ExpiresAt,
		IsAutoRenewEnabled:   sub.IsAutoRenewEnabled,
		MaxWardrobeItems:     plan.MaxWardrobeItems,
		MaxOutfits:           plan.MaxOutfits,
		AiOutfitDailyQuota:   plan.AiOutfitDailyQuota,
		AiChatDailyQuota:     plan.AiChatDailyQuota,
		OutfitRecommendCount: quota.OutfitRecommendCount,
		AiUsageCount:         quota.AiUsageCount,
		LastResetDate:        quota.LastResetDate,
	}
}

func buildUserSubscriptionOverviewDTO(sub *entities.UserSubscription, plan *entities.SubscriptionPlan) *contract.UserSubscriptionOverviewDTO {
	return &contract.UserSubscriptionOverviewDTO{
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanSlug:           plan.Slug,
		ExpiresAt:          sub.ExpiresAt,
		IsAutoRenewEnabled: sub.IsAutoRenewEnabled,
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
	}
}
