package purchase

import (
	"context"
	"time"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type SubscriptionPurchaseUseCase struct {
	walletRepo        repositories.IUserWalletRepository
	depositTxRepo     repositories.IDepositTransactionRepository
	statementRepo     repositories.IWalletStatementRepository
	planRepo          repositories.ISubscriptionPlanRepository
	userSubRepo       repositories.IUserSubscriptionRepository
	eventRepo         repositories.IUserSubscriptionEventRepository
	paymentGateway    payment.IPaymentGatewayService
	eventPublisher    event.IEventPublisher
	uow               shared_repos.IUnitOfWork
	cfg               *config.Config
	notifContract     notificationContract.INotificationContract
	userBriefContract identity_contract.IUserBriefContract
	overviewCache     subscriptioncaching.ISubscriptionOverviewCache
}

func NewSubscriptionPurchaseUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	eventRepo repositories.IUserSubscriptionEventRepository,
	eventPublisher event.IEventPublisher,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	notifContract notificationContract.INotificationContract,
	userBriefContract identity_contract.IUserBriefContract,
	overviewCache subscriptioncaching.ISubscriptionOverviewCache,
) uc_interfaces.ISubscriptionPurchaseUseCase {
	return &SubscriptionPurchaseUseCase{
		walletRepo:        walletRepo,
		depositTxRepo:     depositTxRepo,
		statementRepo:     statementRepo,
		planRepo:          planRepo,
		userSubRepo:       userSubRepo,
		eventRepo:         eventRepo,
		eventPublisher:    eventPublisher,
		paymentGateway:    paymentGateway,
		uow:               uow,
		cfg:               cfg,
		notifContract:     notifContract,
		userBriefContract: userBriefContract,
		overviewCache:     overviewCache,
	}
}

func (uc *SubscriptionPurchaseUseCase) getOrInitLockedSubscriptionForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, bool, error) {
	sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, subscriptionerrors.ErrCurrentSubscriptionLoadFailed()
	}
	if sub != nil {
		return sub, false, nil
	}
	free, err := uc.planRepo.GetDefaultPlan(txCtx)
	if err != nil || free == nil {
		return nil, false, subscriptionerrors.ErrDefaultPlanLoadFailed()
	}
	seed := &entities.UserSubscription{
		UserID:                 userID,
		SubscriptionPlanID:     free.ID,
		SubscriptionPlan:       free,
		CurrentPlanCode:        free.Slug,
		CurrentTierRank:        free.TierRank,
		CurrentPlanKind:        free.PlanKind,
		CurrentBenefitSnapshot: entities.JSONDocument(`{}`),
		StartedAt:              now,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := uc.userSubRepo.ProvisionDefault(txCtx, seed); err != nil {
		return nil, false, err
	}
	sub, err = uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, err
	}
	return sub, false, nil
}

func (uc *SubscriptionPurchaseUseCase) getOrInitLockedWalletForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserWallet, bool, error) {
	wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, subscriptionerrors.ErrQueryWalletBalanceFailed()
	}
	if wallet != nil {
		return wallet, false, nil
	}
	return &entities.UserWallet{
		UserID:    userID,
		Balance:   sharedmoney.Zero,
		Currency:  currency.VND,
		CreatedAt: now,
		UpdatedAt: now,
	}, true, nil
}

func (uc *SubscriptionPurchaseUseCase) persistSubscriptionForPurchase(txCtx context.Context, sub *entities.UserSubscription, isNewSub bool) error {
	if isNewSub {
		if err := uc.userSubRepo.Create(txCtx, sub); err != nil {
			return subscriptionerrors.ErrActivateNewSubscriptionFailed()
		}
		return nil
	}
	if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
		return subscriptionerrors.ErrUpdateSubscriptionExpiryFailed()
	}
	return nil
}

func (uc *SubscriptionPurchaseUseCase) validatePurchase(
	ctx context.Context,
	userID uuid.UUID,
	targetPlan *entities.SubscriptionPlan,
	currentSub *entities.UserSubscription,
	now time.Time,
) error {
	return uc.validatePurchaseWithPendingCheck(ctx, userID, targetPlan, currentSub, now, true)
}

func (uc *SubscriptionPurchaseUseCase) validatePurchaseWithPendingCheck(
	ctx context.Context,
	userID uuid.UUID,
	targetPlan *entities.SubscriptionPlan,
	currentSub *entities.UserSubscription,
	now time.Time,
	checkPending bool,
) error {
	if !targetPlan.IsActive {
		return subscriptionerrors.ErrPlanInactive()
	}

	if checkPending {
		hasPending, err := uc.depositTxRepo.HasPendingDirectPurchase(ctx, userID)
		if err != nil {
			return err
		}
		if hasPending {
			return subscriptionerrors.ErrPendingPaymentExists()
		}
	}

	if currentSub == nil {
		return nil
	}

	currentPlan := currentSub.SubscriptionPlan
	if currentPlan == nil {
		var err error
		currentPlan, err = uc.planRepo.GetByID(ctx, currentSub.SubscriptionPlanID)
		if err != nil {
			return subscriptionerrors.ErrCurrentSubscriptionLoadFailed()
		}
		if currentPlan == nil {
			return subscriptionerrors.ErrCurrentSubscriptionNotFound()
		}
	}

	currentStillValid := currentSub.CurrentPlanKind != 1 || (currentSub.ExpiresAt != nil && currentSub.ExpiresAt.After(now))
	if currentStillValid && currentSub.CurrentPlanKind == 2 && currentSub.CurrentTierRank == targetPlan.TierRank {
		return subscriptionerrors.ErrAlreadyRegisteredUnlimitedPlan()
	}

	if currentStillValid && currentSub.CurrentPlanKind != 0 && targetPlan.TierRank < currentSub.CurrentTierRank {
		return subscriptionerrors.ErrSubscriptionStillActive()
	}

	return nil
}
