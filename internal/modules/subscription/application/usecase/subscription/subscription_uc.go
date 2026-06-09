package subscription

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
	stateSupport  *SubscriptionStateSupport

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
		stateSupport:  NewSubscriptionStateSupport(userSubRepo, planRepo, quotaRepo),
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
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	quota, err := uc.stateSupport.GetOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return BuildUserSubscriptionDTO(sub, plan, quota), nil
}

// GetUserSubscriptionOverview loads ONLY subscription details without high-frequency daily quota metrics
func (uc *SubscriptionUseCase) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionOverviewDTO, error) {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return BuildUserSubscriptionOverviewDTO(sub, plan), nil
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
	planByID, err := uc.stateSupport.ResolveSubscriptionPlans(ctx, subs)
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

		result[sub.UserID] = BuildUserSubscriptionOverviewDTO(sub, plan)
		foundUserIDs[sub.UserID] = struct{}{}
	}

	var missingUserIDs []uuid.UUID
	for _, userID := range userIDs {
		if _, exists := foundUserIDs[userID]; !exists {
			missingUserIDs = append(missingUserIDs, userID)
		}
	}

	if len(missingUserIDs) > 0 {
		defaultPlan, err := uc.planRepo.GetDefaultPlan(ctx)
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
				SubscriptionPlanID: defaultPlan.ID,
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
			result[sub.UserID] = BuildUserSubscriptionOverviewDTO(sub, defaultPlan)
		}
	}

	return result, nil
}
