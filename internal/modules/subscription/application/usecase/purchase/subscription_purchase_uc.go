package purchase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/errorutils"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type SubscriptionPurchaseUseCase struct {
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	eventRepo      repositories.IUserSubscriptionEventRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
	cfg            *config.Config
}

func NewSubscriptionPurchaseUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	eventRepo repositories.IUserSubscriptionEventRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.ISubscriptionPurchaseUseCase {
	return &SubscriptionPurchaseUseCase{
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		planRepo:       planRepo,
		userSubRepo:    userSubRepo,
		eventRepo:      eventRepo,
		paymentGateway: paymentGateway,
		uow:            uow,
		cfg:            cfg,
	}
}

func (uc *SubscriptionPurchaseUseCase) CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error) {
	plan, err := uc.planRepo.GetBySlug(ctx, req.PlanSlug)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrRequestedPlanNotFound()
	}

	// Prevent users from checking out free plans directly.
	if !plan.Price.GreaterThan(sharedmoney.Zero) {
		return nil, subscriptionerrors.ErrFreePlanDirectPurchase()
	}

	if active, err := uc.depositTxRepo.GetActiveDirectPurchase(ctx, userID); err != nil {
		return nil, err
	} else if active != nil {
		url := ""
		if active.PaymentUrl != nil {
			url = *active.PaymentUrl
		}
		return &dto.PaymentLinkDTO{PaymentUrl: url, OrderCode: active.OrderCode, PaymentStatus: active.Status, ExpiresAt: active.ExpiresAt, NextReconciliationAt: active.NextReconciliationAt}, nil
	}

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	snapshotBytes, err := json.Marshal(map[string]any{"maxWardrobeItems": plan.MaxWardrobeItems, "maxOutfits": plan.MaxOutfits, "aiOutfitDailyQuota": plan.AiOutfitDailyQuota, "aiChatDailyQuota": plan.AiChatDailyQuota})
	if err != nil {
		return nil, subscriptionerrors.ErrDirectPurchaseCreateFailed()
	}
	expiresAt := now.Add(time.Duration(uc.cfg.PayOS.ExpiredMinutes) * time.Minute)
	nextReconcile := now.Add(30 * time.Second)
	tx := &entities.DepositTransaction{}
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if err := uc.validatePurchase(txCtx, userID, plan, sub, now); err != nil {
			return err
		}
		planCode, planName, tierRank, kind, pricingVersion := plan.Slug, plan.Name, plan.TierRank, plan.PlanKind, plan.PricingVersion
		tx = &entities.DepositTransaction{UserID: userID, Amount: plan.Price, ExpectedAmount: plan.Price, Currency: currency.VND, Status: depositstatus.Creating, TransactionType: deposittransactiontype.DirectPurchase, SubscriptionPlanID: &plan.ID, PlanCode: &planCode, PlanName: &planName, TierRank: &tierRank, PlanKind: &kind, PurchasedDurationDays: plan.DurationDays, BenefitSnapshot: entities.JSONDocument(snapshotBytes), PricingVersion: &pricingVersion, ExpiresAt: &expiresAt, NextReconciliationAt: &nextReconcile}
		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "ux_active_direct_purchase_per_user") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return subscriptionerrors.ErrPendingPaymentExists()
			}
			return subscriptionerrors.ErrDirectPurchaseCreateFailed()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	returnURL := req.ReturnUrl
	if returnURL == "" {
		returnURL = uc.cfg.PayOS.ReturnUrl
	}
	cancelURL := req.CancelUrl
	if cancelURL == "" {
		cancelURL = uc.cfg.PayOS.CancelUrl
	}
	description := fmt.Sprintf("Purchase plan %s", strings.ReplaceAll(plan.Name, " ", ""))
	if len(description) > 25 {
		description = description[:25]
	}
	result, gatewayErr := uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{OrderCode: tx.OrderCode, Amount: tx.Amount, Description: description, ReturnUrl: returnURL, CancelUrl: cancelURL, ExpiresAt: expiresAt})

	applyErr := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if locked.Status != depositstatus.Creating {
			return nil
		}
		if gatewayErr == nil && result != nil && result.Outcome == payment.OutcomeSucceeded {
			locked.Status = depositstatus.Pending
			locked.PaymentUrl = &result.CheckoutURL
			if result.PaymentLinkID != "" {
				locked.PaymentLinkID = &result.PaymentLinkID
			}
		} else if result != nil && result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			locked.Status = depositstatus.CreationFailed
			locked.FailureReason = &result.ErrorCode
		} else {
			locked.Status = depositstatus.ReconciliationRequired
			locked.ReconciliationAttempts++
			if result != nil {
				locked.LastProviderErrorCode = &result.ErrorCode
			}
			t := now
			locked.LastProviderErrorAt = &t
		}
		return uc.depositTxRepo.Update(txCtx, locked)
	})
	if applyErr != nil {
		return nil, applyErr
	}
	if gatewayErr != nil && result != nil && result.Outcome == payment.OutcomeKnownFailure {
		return nil, errorutils.WrapError(gatewayErr, "Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
	}
	url := ""
	responseStatus := depositstatus.ReconciliationRequired
	if result != nil {
		url = result.CheckoutURL
		if result.Outcome == payment.OutcomeSucceeded {
			responseStatus = depositstatus.Pending
		} else if result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			responseStatus = depositstatus.CreationFailed
		}
	}
	return &dto.PaymentLinkDTO{PaymentUrl: url, OrderCode: tx.OrderCode, PaymentStatus: responseStatus, ExpiresAt: &expiresAt, NextReconciliationAt: &nextReconcile}, nil
}

func (uc *SubscriptionPurchaseUseCase) PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error {
	// Retrieve the subscription plan details by slug.
	plan, err := uc.planRepo.GetBySlug(ctx, planSlug)
	if err != nil {
		return subscriptionerrors.ErrSearchPlanFailed()
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound()
	}

	// Execute the purchase using a database transaction.
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)

		// Fetch and lock the user's subscription record (FOR UPDATE) to prevent race conditions.
		sub, isNewSub, err := uc.getOrInitLockedSubscriptionForPurchase(txCtx, userID, now)
		if err != nil {
			return err
		}

		// Validate the purchase request using the locked subscription row.
		var lockedSub *entities.UserSubscription
		if !isNewSub {
			lockedSub = sub
		}
		if err := uc.validatePurchase(txCtx, userID, plan, lockedSub, now); err != nil {
			return err
		}

		snapshotBytes, err := json.Marshal(map[string]any{
			"maxWardrobeItems":   plan.MaxWardrobeItems,
			"maxOutfits":         plan.MaxOutfits,
			"aiOutfitDailyQuota": plan.AiOutfitDailyQuota,
			"aiChatDailyQuota":   plan.AiChatDailyQuota,
		})
		if err != nil {
			return err
		}
		snapshot := entities.JSONDocument(snapshotBytes)

		// Resolve effective and transition
		freePlan, err := uc.planRepo.GetDefaultPlan(txCtx)
		if err != nil {
			return err
		}
		effective, err := subscription.ResolveEffectiveSubscription(sub, freePlan, now)
		if err != nil {
			return err
		}

		paymentSnapshot := subscription.PaymentSnapshot{
			PlanID:          plan.ID.String(),
			PlanCode:        plan.Slug,
			TierRank:        plan.TierRank,
			PlanKind:        plan.PlanKind,
			DurationDays:    plan.DurationDays,
			BenefitSnapshot: snapshot,
		}

		transition := subscription.EvaluateSubscriptionTransition(subscription.CreateValidation, effective, paymentSnapshot)

		// Determine domain method to call
		var subEvent *entities.UserSubscriptionEvent
		switch transition {
		case subscription.ActivateFinite, subscription.ActivateLifetime:
			subEvent, err = sub.Activate(plan, snapshot, "", now)
		case subscription.ExtendFinite:
			subEvent, err = sub.Extend(plan, snapshot, "", now)
		case subscription.UpgradeFinite, subscription.UpgradeLifetime:
			subEvent, err = sub.Upgrade(plan, snapshot, "", now)
		case subscription.OverlayLifetimeWithFinite:
			subEvent, err = sub.OverlayLifetime(plan, snapshot, "", now)
		default:
			return subscriptionerrors.ErrSubscriptionStillActive()
		}
		if err != nil {
			return err
		}

		// Wallet debit if not free
		if !plan.Price.IsZero() {
			// Fetch and lock the user's wallet record (FOR UPDATE) to guarantee atomic balance deduction.
			walletEntity, _, err := uc.getOrInitLockedWalletForPurchase(txCtx, userID, now)
			if err != nil {
				return err
			}

			// Validate wallet balance beforehand
			if walletEntity.Balance.LessThan(plan.Price) {
				return subscriptionerrors.ErrWalletInsufficientBalance()
			}

			// Call ProcessWalletTransaction
			metadata := wallet.WalletStatementMetadata{
				SourcePlanCode:             &plan.Slug,
				SourceTierRank:             &plan.TierRank,
				ActiveTierRankAtCompletion: &plan.TierRank,
				RenewalAttemptKey:          nil,
			}
			description := fmt.Sprintf("Đăng ký gói hội viên %s thành công qua ví nội bộ", plan.Name)
			if err := wallet.ProcessWalletTransaction(
				txCtx,
				uc.walletRepo,
				uc.statementRepo,
				userID,
				plan.Price.Neg(),
				walletstatementtype.SubscriptionPurchase,
				description,
				nil,
				metadata,
				now,
			); err != nil {
				return err
			}
		}

		// Persist subscription
		if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
			return err
		}

		// Persist subEvent
		if subEvent != nil {
			if err := uc.eventRepo.Create(txCtx, subEvent); err != nil {
				return err
			}
		}

		return nil
	})
}

// getOrInitLockedSubscriptionForPurchase locks the user subscription row in the database.
// If it doesn't exist, it prepares a new subscription entity.
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
	seed := &entities.UserSubscription{UserID: userID, SubscriptionPlanID: free.ID, SubscriptionPlan: free, CurrentPlanCode: free.Slug, CurrentTierRank: free.TierRank, CurrentPlanKind: free.PlanKind, CurrentBenefitSnapshot: entities.JSONDocument(`{}`), StartedAt: now, CreatedAt: now, UpdatedAt: now}
	if err := uc.userSubRepo.ProvisionDefault(txCtx, seed); err != nil {
		return nil, false, err
	}
	sub, err = uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, err
	}
	return sub, false, nil
}

// getOrInitLockedWalletForPurchase locks the user wallet row in the database.
// If it doesn't exist, it prepares a new wallet entity with zero balance.
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

// persistSubscriptionForPurchase saves the modified user subscription into the repository.
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

// validatePurchase checks shared business rules before a user purchases a subscription plan.
// It rejects inactive target plans, conflicting pending purchases, unsafe downgrades,
// and duplicate purchases of permanent plans.
func (uc *SubscriptionPurchaseUseCase) validatePurchase(
	ctx context.Context,
	userID uuid.UUID,
	targetPlan *entities.SubscriptionPlan,
	currentSub *entities.UserSubscription,
	now time.Time,
) error {
	if !targetPlan.IsActive {
		return subscriptionerrors.ErrPlanInactive()
	}

	hasPending, err := uc.depositTxRepo.HasPendingDirectPurchase(ctx, userID)
	if err != nil {
		return err
	}
	if hasPending {
		return subscriptionerrors.ErrPendingPaymentExists()
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
