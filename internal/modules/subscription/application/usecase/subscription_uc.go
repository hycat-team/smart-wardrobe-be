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

