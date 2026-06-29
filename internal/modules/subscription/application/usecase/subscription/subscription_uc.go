package subscription

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type SubscriptionUseCase struct {
	uow                shared_repos.IUnitOfWork
	userSubRepo        repositories.IUserSubscriptionRepository
	planRepo           repositories.ISubscriptionPlanRepository
	walletRepo         repositories.IUserWalletRepository
	statementRepo      repositories.IWalletStatementRepository
	quotaRepo          repositories.IUserDailyQuotaRepository
	renewalAttemptRepo repositories.ISubscriptionRenewalAttemptRepository
	eventRepo          repositories.IUserSubscriptionEventRepository
	cfg                *config.Config
	log                logger.Interface
	stateSupport       *SubscriptionStateSupport

	planContract  contract.ISubscriptionPlanContract
	quotaContract contract.IUserQuotaContract
}

// NewSubscriptionUseCase builds the subscription application service and its state helpers.
func NewSubscriptionUseCase(
	uow shared_repos.IUnitOfWork,
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	renewalAttemptRepo repositories.ISubscriptionRenewalAttemptRepository,
	eventRepo repositories.IUserSubscriptionEventRepository,
	cfg *config.Config,
	log logger.Interface,
	planContract contract.ISubscriptionPlanContract,
	quotaContract contract.IUserQuotaContract,
) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		uow:                uow,
		userSubRepo:        userSubRepo,
		planRepo:           planRepo,
		walletRepo:         walletRepo,
		statementRepo:      statementRepo,
		quotaRepo:          quotaRepo,
		renewalAttemptRepo: renewalAttemptRepo,
		eventRepo:          eventRepo,
		cfg:                cfg,
		log:                log,
		stateSupport:       NewSubscriptionStateSupport(userSubRepo, planRepo, quotaRepo),
		planContract:       planContract,
		quotaContract:      quotaContract,
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

	return mapper.MapToSubscriptionPlanDTOList(plans), nil
}

func (uc *SubscriptionUseCase) SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error) {
	var result bool
	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if sub == nil {
			return subscriptionerrors.ErrUserSubscriptionNotFound()
		}

		if sub.IsAutoRenewEnabled == enable {
			result = sub.IsAutoRenewEnabled
			return nil
		}

		if sub.CurrentPlanKind != plankind.Finite {
			return subscriptionerrors.ErrSubscriptionPlanNotFinite()
		}

		subEvent, err := sub.SetAutoRenew(enable, "", now)
		if err != nil {
			return err
		}

		if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
			return err
		}

		if subEvent != nil {
			if err := uc.eventRepo.Create(txCtx, subEvent); err != nil {
				return err
			}
		}

		result = sub.IsAutoRenewEnabled
		return nil
	})
	if err != nil {
		return false, err
	}
	return result, nil
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

	return mapper.BuildUserSubscriptionDTO(sub, plan, quota), nil
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

	return mapper.BuildUserSubscriptionOverviewDTO(sub, plan), nil
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
	defaultPlan, defaultPlanErr := uc.planRepo.GetDefaultPlan(ctx)
	if defaultPlanErr != nil || defaultPlan == nil {
		return nil, subscriptionerrors.ErrDefaultPlanLoadFailed()
	}
	for _, sub := range subs {
		if sub == nil {
			continue
		}

		plan := planByID[sub.SubscriptionPlanID]
		if sub.CurrentPlanKind == 1 && sub.ExpiresAt != nil && !sub.ExpiresAt.After(time.Now()) {
			if sub.FallbackPlanID != nil {
				plan = planByID[*sub.FallbackPlanID]
			} else {
				plan = defaultPlan
			}
		}
		if plan == nil {
			return nil, subscriptionerrors.ErrSubscriptionPlanNotFound()
		}

		result[sub.UserID] = mapper.BuildUserSubscriptionOverviewDTO(sub, plan)
		foundUserIDs[sub.UserID] = struct{}{}
	}

	var missingUserIDs []uuid.UUID
	for _, userID := range userIDs {
		if _, exists := foundUserIDs[userID]; !exists {
			missingUserIDs = append(missingUserIDs, userID)
		}
	}

	if len(missingUserIDs) > 0 {
		newSubs := make([]*entities.UserSubscription, 0, len(missingUserIDs))
		for _, userID := range missingUserIDs {
			newSub := &entities.UserSubscription{
				UserID:                 userID,
				SubscriptionPlanID:     defaultPlan.ID,
				SubscriptionPlan:       defaultPlan,
				CurrentPlanCode:        defaultPlan.Slug,
				CurrentTierRank:        defaultPlan.TierRank,
				CurrentPlanKind:        defaultPlan.PlanKind,
				CurrentBenefitSnapshot: entities.JSONDocument(`{}`),
				StartedAt:              time.Now(),
				IsAutoRenewEnabled:     false,
			}
			newSubs = append(newSubs, newSub)
		}

		if err := uc.userSubRepo.BulkCreate(ctx, newSubs); err != nil {
			return nil, err
		}

		for _, sub := range newSubs {
			result[sub.UserID] = mapper.BuildUserSubscriptionOverviewDTO(sub, defaultPlan)
		}
	}

	return result, nil
}
