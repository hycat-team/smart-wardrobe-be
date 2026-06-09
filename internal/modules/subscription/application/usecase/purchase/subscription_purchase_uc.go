package purchase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/errorutils"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SubscriptionPurchaseUseCase struct {
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
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
		paymentGateway: paymentGateway,
		uow:            uow,
		cfg:            cfg,
	}
}

func (uc *SubscriptionPurchaseUseCase) CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error) {
	// Query the plan details by slug to check if it exists.
	plan, err := uc.planRepo.GetBySlug(ctx, req.PlanSlug)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrRequestedPlanNotFound
	}

	// Prevent users from checking out free plans directly.
	if !plan.Price.GreaterThan(sharedmoney.Zero) {
		return nil, subscriptionerrors.ErrFreePlanDirectPurchase
	}

	// Fetch existing user subscription to perform validation.
	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate the purchase request against business rules.
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	if err := uc.validatePurchase(ctx, userID, plan, sub, now); err != nil {
		return nil, err
	}

	var checkoutURL string
	var orderCode int64

	// Define a transaction function to create a pending payment record and generate the payment link.
	createDirectPurchase := func(txCtx context.Context) error {
		// Initialize the deposit/payment transaction entity with PENDING status.
		tx := &entities.DepositTransaction{
			UserID:             userID,
			Amount:             plan.Price,
			Currency:           currency.VND,
			Status:             depositstatus.Pending,
			TransactionType:    deposittransactiontype.DirectPurchase,
			SubscriptionPlanID: &plan.ID,
			OrderCode:          timeutils.GenerateOrderCode(), // Custom application-generated unique numeric identifier
			PaymentUrl:         nil,
		}

		// Insert the transaction into the database first.
		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return subscriptionerrors.ErrDirectPurchaseCreateFailed
		}

		// Configure return/cancel callback URLs for the payment gateway.
		returnURL := req.ReturnUrl
		if returnURL == "" {
			returnURL = uc.cfg.PayOS.ReturnUrl
		}
		cancelURL := req.CancelUrl
		if cancelURL == "" {
			cancelURL = uc.cfg.PayOS.CancelUrl
		}

		// Normalize descriptive text for the bank checkout session.
		normalizedPlanName := strings.ReplaceAll(plan.Name, " ", "")
		description := fmt.Sprintf("Purchase plan %s", normalizedPlanName)
		if len(description) > 25 {
			description = description[:25] // Standard length limit for payment gateways (e.g., PayOS)
		}

		// Call the external payment gateway service to initiate checkout.
		checkoutURL, err = uc.paymentGateway.CreateCheckoutSession(txCtx, &payment.CheckoutSessionReq{
			OrderCode:   tx.OrderCode,
			Amount:      tx.Amount,
			Description: description,
			ReturnUrl:   returnURL,
			CancelUrl:   cancelURL,
		})
		if err != nil {
			return errorutils.WrapError(err, "Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
		}

		// Update the transaction in database with the generated checkout payment URL.
		tx.PaymentUrl = &checkoutURL
		if err := uc.depositTxRepo.Update(txCtx, tx); err != nil {
			return subscriptionerrors.ErrPaymentLinkCreateFailed
		}

		orderCode = tx.OrderCode
		return nil
	}

	// Execute the entire setup workflow in a single database transaction.
	if err := uc.uow.Execute(ctx, createDirectPurchase); err != nil {
		return nil, err
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutURL,
		OrderCode:  orderCode,
	}, nil
}

func (uc *SubscriptionPurchaseUseCase) PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error {
	// Retrieve the subscription plan details by slug.
	plan, err := uc.planRepo.GetBySlug(ctx, planSlug)
	if err != nil {
		return subscriptionerrors.ErrSearchPlanFailed
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound
	}

	// Execute the purchase using a database transaction.
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		description := fmt.Sprintf("Đăng ký gói hội viên %s thành công qua ví nội bộ", plan.Name)

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

		// Special case - if the plan is free, apply the plan immediately without wallet debit.
		if plan.Price.IsZero() {
			uc.applyPlanToSubscriptionEntity(sub, isNewSub, plan, now)
			if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
				return err
			}
			return nil
		}

		// Fetch and lock the user's wallet record (FOR UPDATE) to guarantee atomic balance deduction.
		wallet, isNewWallet, err := uc.getOrInitLockedWalletForPurchase(txCtx, userID, now)
		if err != nil {
			return err
		}

		// Debit the wallet balance and persist the changes.
		prevBalance, err := uc.applyWalletDebitToLockedWallet(txCtx, wallet, isNewWallet, plan.Price, now)
		if err != nil {
			return err
		}

		// Record a wallet transaction statement (ledger) for auditing purposes.
		if err := uc.createPurchaseWalletStatement(txCtx, userID, plan.Price, prevBalance, wallet.Balance, description); err != nil {
			return err
		}

		// Calculate expiration/extension, update subscription state, and save the subscription record.
		uc.applyPlanToSubscriptionEntity(sub, isNewSub, plan, now)
		if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
			return err
		}

		return nil
	})
}

// getOrInitLockedSubscriptionForPurchase locks the user subscription row in the database.
// If it doesn't exist, it prepares a new subscription entity.
func (uc *SubscriptionPurchaseUseCase) getOrInitLockedSubscriptionForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, bool, error) {
	sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, subscriptionerrors.ErrCurrentSubscriptionLoadFailed
	}
	if sub != nil {
		return sub, false, nil
	}

	return &entities.UserSubscription{
		UserID:    userID,
		CreatedAt: now,
	}, true, nil
}

// getOrInitLockedWalletForPurchase locks the user wallet row in the database.
// If it doesn't exist, it prepares a new wallet entity with zero balance.
func (uc *SubscriptionPurchaseUseCase) getOrInitLockedWalletForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserWallet, bool, error) {
	wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, subscriptionerrors.ErrQueryWalletBalanceFailed
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

// applyWalletDebitToLockedWallet validates the balance and subtracts the plan price.
// Persists the change to the repository.
func (uc *SubscriptionPurchaseUseCase) applyWalletDebitToLockedWallet(txCtx context.Context, wallet *entities.UserWallet, isNewWallet bool, amount decimal.Decimal, now time.Time) (decimal.Decimal, error) {
	if wallet.Balance.LessThan(amount) {
		return sharedmoney.Zero, subscriptionerrors.ErrWalletInsufficientBalance
	}

	prevBalance := wallet.Balance
	wallet.Balance = wallet.Balance.Sub(amount)
	wallet.UpdatedAt = now

	if isNewWallet {
		if err := uc.walletRepo.Create(txCtx, wallet); err != nil {
			return sharedmoney.Zero, subscriptionerrors.ErrWalletCreateFailed
		}
	} else {
		if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
			return sharedmoney.Zero, subscriptionerrors.ErrWalletBalanceUpdateFailed
		}
	}

	return prevBalance, nil
}

// createPurchaseWalletStatement creates a negative balance change audit ledger entry.
func (uc *SubscriptionPurchaseUseCase) createPurchaseWalletStatement(txCtx context.Context, userID uuid.UUID, amount decimal.Decimal, prevBalance decimal.Decimal, newBalance decimal.Decimal, description string) error {
	statement := &entities.WalletStatement{
		UserID:          userID,
		Amount:          amount.Neg(), // Negative amount indicates a deduction/debit
		TransactionType: walletstatementtype.SubscriptionPurchase,
		PreviousBalance: prevBalance,
		NewBalance:      newBalance,
		Description:     description,
	}

	if err := uc.statementRepo.Create(txCtx, statement); err != nil {
		return subscriptionerrors.ErrWalletStatementSaveFailed
	}

	return nil
}

// applyPlanToSubscriptionEntity computes subscription validity duration and sets update values.
func (uc *SubscriptionPurchaseUseCase) applyPlanToSubscriptionEntity(sub *entities.UserSubscription, isNewSub bool, plan *entities.SubscriptionPlan, now time.Time) {
	var expiresAt *time.Time

	if plan.DurationDays == nil {
		expiresAt = nil
	} else {
		days := *plan.DurationDays
		var expiry time.Time
		// If user extends the SAME plan that is currently active, stack/add to the expiration time.
		// If switching plans or the previous plan expired, start fresh duration from now.
		if !isNewSub && sub.IsActive && sub.ExpiresAt != nil && sub.ExpiresAt.After(now) && sub.SubscriptionPlanID == plan.ID {
			expiry = sub.ExpiresAt.AddDate(0, 0, days)
		} else {
			expiry = now.AddDate(0, 0, days)
		}
		expiresAt = &expiry
	}

	sub.SubscriptionPlanID = plan.ID
	sub.ExpiresAt = expiresAt
	sub.IsActive = true
	sub.UpdatedAt = now
}

// persistSubscriptionForPurchase saves the modified user subscription into the repository.
func (uc *SubscriptionPurchaseUseCase) persistSubscriptionForPurchase(txCtx context.Context, sub *entities.UserSubscription, isNewSub bool) error {
	if isNewSub {
		if err := uc.userSubRepo.Create(txCtx, sub); err != nil {
			return subscriptionerrors.ErrActivateNewSubscriptionFailed
		}
		return nil
	}

	if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
		return subscriptionerrors.ErrUpdateSubscriptionExpiryFailed
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
		return subscriptionerrors.ErrPlanInactive
	}

	hasPending, err := uc.depositTxRepo.HasPendingDirectPurchase(ctx, userID)
	if err != nil {
		return err
	}
	if hasPending {
		return subscriptionerrors.ErrPendingPaymentExists
	}

	if currentSub == nil || !currentSub.IsActive {
		return nil
	}

	currentPlan := currentSub.SubscriptionPlan
	if currentPlan == nil {
		var err error
		currentPlan, err = uc.planRepo.GetByID(ctx, currentSub.SubscriptionPlanID)
		if err != nil {
			return subscriptionerrors.ErrCurrentSubscriptionLoadFailed
		}
		if currentPlan == nil {
			return subscriptionerrors.ErrCurrentSubscriptionNotFound
		}
	}

	if currentSub.SubscriptionPlanID == targetPlan.ID && currentSub.ExpiresAt == nil {
		return subscriptionerrors.ErrAlreadyRegisteredUnlimitedPlan
	}

	currentIsPaid := currentPlan.Price.GreaterThan(sharedmoney.Zero)
	currentStillValid := currentSub.ExpiresAt == nil || currentSub.ExpiresAt.After(now)
	targetIsLower := targetPlan.Price.LessThan(currentPlan.Price)

	if currentIsPaid && currentStillValid && targetIsLower {
		return subscriptionerrors.ErrSubscriptionStillActive
	}

	return nil
}

