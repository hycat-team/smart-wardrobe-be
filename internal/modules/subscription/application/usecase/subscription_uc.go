package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type SubscriptionUseCase struct {
	uow           shared_repos.IUnitOfWork
	userSubRepo   repositories.IUserSubscriptionRepository
	planRepo      repositories.ISubscriptionPlanRepository
	walletRepo    repositories.IUserWalletRepository
	statementRepo repositories.IWalletStatementRepository
	quotaRepo     repositories.IUserDailyQuotaRepository
	cfg           *config.Config
	log           logger.Interface

	planContract  contract.ISubscriptionPlanContract
	quotaContract contract.IUserQuotaContract
}

func NewSubscriptionUseCase(
	uow shared_repos.IUnitOfWork,
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	cfg *config.Config,
	log logger.Interface,
	planContract contract.ISubscriptionPlanContract,
	quotaContract contract.IUserQuotaContract,
) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		uow:           uow,
		userSubRepo:   userSubRepo,
		planRepo:      planRepo,
		walletRepo:    walletRepo,
		statementRepo: statementRepo,
		quotaRepo:     quotaRepo,
		cfg:           cfg,
		log:           log,
		planContract:  planContract,
		quotaContract: quotaContract,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	return uc.quotaContract.GetAndResetDailyQuota(ctx, userID)
}

func (uc *SubscriptionUseCase) GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtoPlans := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		dtoPlans = append(dtoPlans, &dto.SubscriptionPlanDTO{
			ID:                 plan.ID,
			Slug:               plan.Slug,
			Name:               plan.Name,
			Price:              sharedmoney.ToFloat(plan.Price),
			MaxWardrobeItems:   plan.MaxWardrobeItems,
			MaxOutfits:         plan.MaxOutfits,
			AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
			AiChatDailyQuota:   plan.AiChatDailyQuota,
			DurationDays:       plan.DurationDays,
		})
	}
	return dtoPlans, nil
}

func (uc *SubscriptionUseCase) SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error) {
	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, subscriptionerrors.ErrUserSubscriptionNotFound
	}

	if sub.IsAutoRenewEnabled == enable {
		return sub.IsAutoRenewEnabled, nil
	}

	if enable && sub.ExpiresAt != nil && sub.ExpiresAt.Before(time.Now()) {
		return false, subscriptionerrors.ErrSubscriptionExpiredAutoRenew
	}

	sub.IsAutoRenewEnabled = enable
	sub.UpdatedAt = timeutils.GetNow(uc.cfg.Database.TimeZone)

	err = uc.userSubRepo.Update(ctx, sub)
	if err != nil {
		return false, err
	}

	return sub.IsAutoRenewEnabled, nil
}

// GetUserSubscription loads subscription details and daily quotas aggregated from multiple tables
func (uc *SubscriptionUseCase) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	quota, err := uc.getOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, subscriptionerrors.ErrSubscriptionPlanNotFound
		}
		plan = p
	}

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
	}, nil
}

// GetUserSubscriptionOverview loads ONLY subscription details without high-frequency daily quota metrics
func (uc *SubscriptionUseCase) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionOverviewDTO, error) {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, subscriptionerrors.ErrSubscriptionPlanNotFound
		}
		plan = p
	}

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
	}, nil
}

func (uc *SubscriptionUseCase) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*contract.UserSubscriptionOverviewDTO, error) {
	result := make(map[uuid.UUID]*contract.UserSubscriptionOverviewDTO, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	subs, err := uc.userSubRepo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// Keep the query pattern batch-safe even if a repository change accidentally drops
	// the SubscriptionPlan preload in the future. We resolve every missing plan in one
	// batched lookup instead of issuing one query per subscription row.
	planByID, err := uc.resolveSubscriptionPlans(ctx, subs)
	if err != nil {
		return nil, err
	}

	foundUserIDs := make(map[uuid.UUID]struct{}, len(subs))
	for _, sub := range subs {
		if sub == nil {
			continue
		}

		plan := planByID[sub.SubscriptionPlanID]
		if plan == nil {
			return nil, subscriptionerrors.ErrSubscriptionPlanNotFound
		}

		result[sub.UserID] = &contract.UserSubscriptionOverviewDTO{
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
		foundUserIDs[sub.UserID] = struct{}{}
	}

	var missingUserIDs []uuid.UUID
	for _, userID := range userIDs {
		if _, exists := foundUserIDs[userID]; !exists {
			missingUserIDs = append(missingUserIDs, userID)
		}
	}

	if len(missingUserIDs) > 0 {
		defaultPlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
		if err != nil {
			return nil, err
		}
		defaultPlan, err := uc.planRepo.GetByID(ctx, defaultPlanID)
		if err != nil {
			return nil, err
		}
		if defaultPlan == nil {
			return nil, subscriptionerrors.ErrSubscriptionPlanNotFound
		}

		newSubs := make([]*entities.UserSubscription, 0, len(missingUserIDs))
		for _, userID := range missingUserIDs {
			newSub := &entities.UserSubscription{
				UserID:             userID,
				SubscriptionPlanID: defaultPlanID,
				SubscriptionPlan:   defaultPlan,
				IsActive:           true,
				IsAutoRenewEnabled: false,
			}
			newSubs = append(newSubs, newSub)
		}

		if err := uc.userSubRepo.BulkCreate(ctx, newSubs); err != nil {
			return nil, err
		}

		for _, sub := range newSubs {
			result[sub.UserID] = &contract.UserSubscriptionOverviewDTO{
				PlanID:             defaultPlan.ID,
				PlanName:           defaultPlan.Name,
				PlanSlug:           defaultPlan.Slug,
				ExpiresAt:          sub.ExpiresAt,
				IsAutoRenewEnabled: sub.IsAutoRenewEnabled,
				MaxWardrobeItems:   defaultPlan.MaxWardrobeItems,
				MaxOutfits:         defaultPlan.MaxOutfits,
				AiOutfitDailyQuota: defaultPlan.AiOutfitDailyQuota,
				AiChatDailyQuota:   defaultPlan.AiChatDailyQuota,
			}
		}
	}

	return result, nil
}

// === Helper ===

func (uc *SubscriptionUseCase) getOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		defaultPlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
		if err != nil {
			return nil, err
		}
		defaultPlan, err := uc.planRepo.GetByID(ctx, defaultPlanID)
		if err != nil {
			return nil, err
		}
		if defaultPlan == nil {
			return nil, subscriptionerrors.ErrDefaultPlanConfigNotFound
		}

		sub = &entities.UserSubscription{
			UserID:             userID,
			SubscriptionPlanID: defaultPlanID,
			IsActive:           true,
			SubscriptionPlan:   defaultPlan,
		}
		err = uc.userSubRepo.Create(ctx, sub)
		if err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (uc *SubscriptionUseCase) resolveSubscriptionPlans(ctx context.Context, subs []*entities.UserSubscription) (map[uuid.UUID]*entities.SubscriptionPlan, error) {
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

	plans, err := uc.planRepo.GetByIDs(ctx, missingPlanIDs)
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

func (uc *SubscriptionUseCase) getOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := uc.quotaRepo.GetByUserID(ctx, userID)
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
		err = uc.quotaRepo.Create(ctx, quota)
		if err != nil {
			return nil, err
		}
	}
	return quota, nil
}
