package subscription

import (
	"context"
	"fmt"
	"time"

	admin_contract "smart-wardrobe-be/services/account/contract/admin"
	"smart-wardrobe-be/services/account/dto/subscription"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

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

	planUseCase   uc_interfaces.ISubscriptionPlanUseCase
	quotaUseCase  uc_interfaces.IUserQuotaUseCase
	notifContract notificationContract.INotificationContract
	overviewCache subscriptioncaching.ISubscriptionOverviewCache
	// 024 T048: every plan change in this use case publishes through one helper.
	planChangePublisher *PlanChangePublisher
	// 024 T083: the batch grant path needs to resolve an administrator-supplied
	// identifier to an account, and to write the audit record. Both are new
	// dependencies rather than extra work for existing ones, which is why the
	// grant logic lives in its own file instead of being appended here.
	adminUserResolver *AdminUserResolver
	auditLogContract  admin_contract.IAuditLogContract
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
	planUseCase uc_interfaces.ISubscriptionPlanUseCase,
	quotaUseCase uc_interfaces.IUserQuotaUseCase,
	notifContract notificationContract.INotificationContract,
	overviewCache subscriptioncaching.ISubscriptionOverviewCache,
	planChangePublisher *PlanChangePublisher,
	adminUserResolver *AdminUserResolver,
	auditLogContract admin_contract.IAuditLogContract,
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
		stateSupport: NewSubscriptionStateSupport(
			userSubRepo, planRepo, quotaRepo, planChangePublisher, uow,
		),
		planUseCase:         planUseCase,
		quotaUseCase:        quotaUseCase,
		notifContract:       notifContract,
		overviewCache:       overviewCache,
		planChangePublisher: planChangePublisher,
		adminUserResolver:   adminUserResolver,
		auditLogContract:    auditLogContract,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error) {
	return uc.quotaUseCase.GetAndResetDailyQuota(ctx, userID)
}

func (uc *SubscriptionUseCase) GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanBriefRes, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return mapper.MapToSubscriptionPlanBriefList(plans), nil
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

		// 024 T051/FR-012: this publishes even though the plan does not change.
		// The receiver is entitled to see every plan-relevant event and to
		// decide for itself that AUTO_RENEW_UPDATED requires no action — that
		// decision belongs downstream, not here. The receiver used to treat
		// this type as a grant-creating change, which is the mirror-image bug:
		// the sender was silent and the receiver guessed.
		plan, err := uc.planRepo.GetByID(txCtx, sub.SubscriptionPlanID)
		if err != nil {
			return err
		}
		if plan == nil {
			return subscriptionerrors.ErrSubscriptionPlanNotFound()
		}
		if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, subEvent, sub, plan); err != nil {
			return err
		}

		result = sub.IsAutoRenewEnabled
		return nil
	})
	if err != nil {
		return false, err
	}
	// 024 T154: this was a second instance of the defect T151 fixed on the admin
	// grant path - a post-commit cache delete on the caller's context, with the
	// error discarded. The transaction has already committed here, so the same two
	// things can go wrong: the client disconnects, or the caller's deadline has
	// passed, and the cached overview keeps serving the old auto-renew flag with
	// nothing anywhere reporting it.
	//
	// Reusing `invalidateOverviewCache` rather than repeating its fix keeps the two
	// paths from drifting again, which is what caused this in the first place.
	uc.invalidateOverviewCache(ctx, userID)
	return result, nil
}

// GetUserSubscription loads subscription details and daily quotas aggregated from multiple tables
func (uc *SubscriptionUseCase) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error) {
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
func (uc *SubscriptionUseCase) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionOverviewRes, error) {
	sub, err := uc.stateSupport.GetOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.stateSupport.LoadPlanForSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	return mapper.BuildUserSubscriptionOverviewRes(sub, plan), nil
}

// EnsureDefaultSubscription provisions the default plan for a brand-new account.
// See SubscriptionStateSupport.EnsureDefaultSubscription for why it runs on the
// caller's transaction rather than opening one of its own.
func (uc *SubscriptionUseCase) EnsureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	return uc.stateSupport.EnsureDefaultSubscription(ctx, userID)
}

func (uc *SubscriptionUseCase) GetUserSubscriptionOverviews(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*dto.UserSubscriptionOverviewRes, error) {
	result := make(map[uuid.UUID]*dto.UserSubscriptionOverviewRes, len(userIDs))
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

		result[sub.UserID] = mapper.BuildUserSubscriptionOverviewRes(sub, plan)
		foundUserIDs[sub.UserID] = struct{}{}
	}

	var missingUserIDs []uuid.UUID
	for _, userID := range userIDs {
		if _, exists := foundUserIDs[userID]; !exists {
			missingUserIDs = append(missingUserIDs, userID)
		}
	}

	if len(missingUserIDs) > 0 {
		// 024 T050 / FR-014: this is a READ path that was creating rows. It
		// provisioned a bare subscription with no history and no event, so
		// every account that happened to be part of a batch read ended up
		// without an AI policy grant and without any record explaining why.
		//
		// The write itself stays here (it cannot move without changing the
		// contract of this endpoint), but it is now wrapped in a transaction
		// that also writes one history row and one outbox record per account, so
		// the accounts provisioned here are indistinguishable from accounts
		// provisioned by any other path.
		provisioned, err := uc.provisionMissingSubscriptions(ctx, missingUserIDs, defaultPlan)
		if err != nil {
			return nil, err
		}
		for _, sub := range provisioned {
			result[sub.UserID] = mapper.BuildUserSubscriptionOverviewRes(sub, defaultPlan)
		}
	}

	return result, nil
}

// provisionMissingSubscriptions creates the default subscription for accounts
// that have none, and publishes a plan-change event for each one.
func (uc *SubscriptionUseCase) provisionMissingSubscriptions(
	ctx context.Context,
	userIDs []uuid.UUID,
	defaultPlan *entities.SubscriptionPlan,
) ([]*entities.UserSubscription, error) {
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	created := make([]*entities.UserSubscription, 0, len(userIDs))

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		created = created[:0]
		for _, userID := range userIDs {
			// The default plan carries no duration, so auto-renew is meaningless
			// and leaving it on would make a later upgrade inherit a flag that
			// was never set for that plan.
			newSub := &entities.UserSubscription{
				UserID:                 userID,
				SubscriptionPlanID:     defaultPlan.ID,
				SubscriptionPlan:       defaultPlan,
				CurrentPlanCode:        defaultPlan.Slug,
				CurrentTierRank:        defaultPlan.TierRank,
				CurrentPlanKind:        defaultPlan.PlanKind,
				CurrentBenefitSnapshot: BuildPlanBenefitSnapshot(defaultPlan),
				StartedAt:              now,
				CreatedAt:              now,
				UpdatedAt:              now,
				IsAutoRenewEnabled:     false,
			}
			if err := uc.userSubRepo.ProvisionDefault(txCtx, newSub); err != nil {
				return err
			}
			// ProvisionDefault may leave the primary key unset depending on the
			// implementation, so the version is read back to keep the event key
			// stable and to carry the persisted id.
			persisted, err := uc.userSubRepo.GetByUserID(txCtx, userID)
			if err != nil {
				return err
			}
			if persisted == nil {
				persisted = newSub
			}
			created = append(created, persisted)

			toPlanCode := persisted.CurrentPlanCode
			toTierRank := persisted.CurrentTierRank
			provisionedEvent := &entities.UserSubscriptionEvent{
				EventKey:    fmt.Sprintf("DEFAULT_PROVISION_BULK:%s:%d", userID, persisted.Version),
				UserID:      userID,
				EventType:   string(subscriptioneventtype.ACTIVATED),
				ToPlanCode:  &toPlanCode,
				ToTierRank:  &toTierRank,
				OccurredAt:  now,
				EffectiveAt: now,
			}
			if _, err := uc.planChangePublisher.PublishPlanChange(txCtx, provisionedEvent, persisted, defaultPlan); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// AdminList lists user subscriptions with optional filtering by planCode and status.
func (uc *SubscriptionUseCase) AdminList(ctx context.Context, planCode *string, status *usersubscriptionstatus.UserSubscriptionStatus, page, limit int) ([]*dto.AdminUserSubscriptionItemDTO, int64, error) {
	filter := repositories.AdminUserSubscriptionFilter{
		PlanCode: planCode,
		Status:   status,
	}
	subs, total, err := uc.userSubRepo.ListAdmin(ctx, filter, page, limit)
	if err != nil {
		return nil, 0, err
	}

	planByID, err := uc.stateSupport.ResolveSubscriptionPlans(ctx, subs)
	if err != nil {
		return nil, 0, err
	}

	return mapper.BuildAdminUserSubscriptionItemDTOList(subs, planByID), total, nil
}

// AdminCancel downgrades a user's subscription back to the default free plan.
//
// 024 T048: this path used to mutate the subscription and stop there. Because
// no history row and no event were written, the AI service never learned about
// the revocation and the account kept its paid entitlements until something
// else happened to refresh them. The revoke now goes through the same publish
// helper as every other plan change, and the receiver is told to close every
// active grant.
func (uc *SubscriptionUseCase) AdminCancel(ctx context.Context, userID uuid.UUID) error {
	defaultPlan, err := uc.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return err
	}
	if defaultPlan == nil {
		return subscriptionerrors.ErrDefaultPlanConfigNotFound()
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if sub == nil {
			return subscriptionerrors.ErrUserSubscriptionNotFound()
		}

		now := time.Now()
		// The behaviour methods own the state transition, including the
		// fallback-field cleanup and the version bump. Reimplementing the
		// field-by-field assignment here is how the two drifted apart before.
		downgradeEvent, err := sub.DowngradeToFree(defaultPlan, "", now)
		if err != nil {
			return err
		}

		if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
			return err
		}

		// Reset quota if needed
		if quota, err := uc.quotaRepo.GetByUserID(txCtx, userID); err == nil && quota != nil {
			quota.OutfitRecommendCount = 0
			quota.AiUsageCount = 0
			quota.LastResetDate = now
			_ = uc.quotaRepo.Update(txCtx, quota)
		}

		_, err = uc.planChangePublisher.PublishPlanChange(txCtx, downgradeEvent, sub, defaultPlan)
		return err
	})
}
