package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/config"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PaymentWebhookUseCase struct {
	cfg            *config.Config
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
	log            logger.Interface
}

func NewPaymentWebhookUseCase(
	cfg *config.Config,
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	log logger.Interface,
) uc_interfaces.IPaymentWebhookUseCase {
	return &PaymentWebhookUseCase{
		cfg:            cfg,
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		planRepo:       planRepo,
		userSubRepo:    userSubRepo,
		paymentGateway: paymentGateway,
		uow:            uow,
		log:            log,
	}
}

func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	// Verify the webhook authenticity by validating its cryptographic signature with the gateway's public keys.
	_, err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature)
	if err != nil {
		return subscriptionerrors.ErrVerifySignatureFailed
	}

	// Parse the webhook callback payload structure.
	var payload struct {
		Code    string `json:"code"`
		Desc    string `json:"desc"`
		Success bool   `json:"success"`
		Data    struct {
			OrderCode              int64       `json:"orderCode"`
			Amount                 json.Number `json:"amount"`
			Description            string      `json:"description"`
			AccountNumber          string      `json:"accountNumber"`
			Reference              string      `json:"reference"`
			TransactionDateTime    string      `json:"transactionDateTime"`
			Currency               string      `json:"currency"`
			PaymentLinkId          string      `json:"paymentLinkId"`
			Code                   string      `json:"code"`
			Desc                   string      `json:"desc"`
			CounterAccountBankId   string      `json:"counterAccountBankId"`
			CounterAccountBankName string      `json:"counterAccountBankName"`
			CounterAccountName     string      `json:"counterAccountName"`
			CounterAccountNumber   string      `json:"counterAccountNumber"`
			VirtualAccountName     string      `json:"virtualAccountName"`
			VirtualAccountNumber   string      `json:"virtualAccountNumber"`
		} `json:"data"`
		Signature string `json:"signature"`
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.UseNumber() // Use json.Number to avoid precision loss on large integer numbers
	if err := decoder.Decode(&payload); err != nil {
		return subscriptionerrors.ErrWebhookPayloadMalformed
	}

	// Check transaction status codes. Code "00" indicates success.
	// If the transaction failed on the gateway side, we just return nil (we don't credit users).
	if payload.Code != "00" || payload.Data.Code != "00" {
		uc.log.Warn("Ignored unsuccessful payment webhook",
			zap.Int64("order_code", payload.Data.OrderCode),
			zap.String("gateway_code", payload.Code),
			zap.String("gateway_data_code", payload.Data.Code),
			zap.String("result", "ignored_non_success"),
		)
		return nil
	}

	// Convert amount string from payload to decimal.
	webhookAmount, err := decimal.NewFromString(payload.Data.Amount.String())
	if err != nil {
		return subscriptionerrors.ErrInvalidAmount
	}
	if err := sharedmoney.ValidateSupportedCurrencyText(payload.Data.Currency); err != nil {
		return err
	}

	// Query the matching pending deposit transaction record by its OrderCode.
	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, payload.Data.OrderCode)
	if err != nil {
		return subscriptionerrors.ErrQueryTransactionFailed
	}
	if tx == nil {
		return subscriptionerrors.ErrTransactionNotFound(payload.Data.OrderCode)
	}
	if err := sharedmoney.ValidateSupportedCurrency(tx.Currency); err != nil {
		return err
	}

	// Verify the received payment amount matches the expected record amount to prevent fraud/underpayment.
	if !webhookAmount.Equal(tx.Amount) {
		return subscriptionerrors.ErrPaymentAmountMismatch
	}

	// Optimistic check - if the transaction is already marked success, skip reprocessing (idempotency).
	if tx.Status == depositstatus.Success {
		return nil
	}

	// Serialize gateway data payload for audit logs.
	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return subscriptionerrors.ErrProcessPaymentGatewayFailed
	}
	detailsStr := string(rawBytes)
	reference := payload.Data.Reference

	// Execute transaction processing workflow within a database transaction.
	processPaymentWebhook := func(txCtx context.Context) error {
		// Re-fetch the transaction record using row-level locking (FOR UPDATE)
		// to guarantee thread safety and prevent double-processing (idempotency guard).
		lockedTx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, payload.Data.OrderCode)
		if err != nil {
			return subscriptionerrors.ErrLockTransactionFailed
		}
		if lockedTx == nil {
			return subscriptionerrors.ErrTransactionNotFound(payload.Data.OrderCode)
		}

		// Double-check the status inside the locked transaction context.
		if lockedTx.Status == depositstatus.Success {
			return nil
		}

		// Update the transaction status to SUCCESS and record references.
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		lockedTx.Status = depositstatus.Success
		lockedTx.GatewayReference = &reference
		lockedTx.GatewayDetails = &detailsStr

		if err := uc.depositTxRepo.Update(txCtx, lockedTx); err != nil {
			return subscriptionerrors.ErrCompletePaymentRecordFailed
		}

		// Route processing based on the TransactionType.
		switch lockedTx.TransactionType {
		case deposittransactiontype.WalletTopup:
			// Top-up flow: Credit the user's wallet.
			if err := uc.executeWalletTopUpWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		case deposittransactiontype.DirectPurchase:
			// Direct purchase flow: Apply/extend the subscription plan directly.
			if err := uc.executeDirectPurchaseWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, processPaymentWebhook); err != nil {
		uc.log.Error("Failed to process payment webhook",
			zap.Int64("order_code", payload.Data.OrderCode),
			zap.String("gateway_reference", payload.Data.Reference),
			zap.String("currency", payload.Data.Currency),
			zap.String("result", "failed"),
			zap.Error(err),
		)
		return err
	}

	uc.log.Info("Processed payment webhook successfully",
		zap.Int64("order_code", payload.Data.OrderCode),
		zap.String("gateway_reference", payload.Data.Reference),
		zap.String("currency", payload.Data.Currency),
		zap.String("result", "processed"),
	)
	return nil
}

func (uc *PaymentWebhookUseCase) executeWalletTopUpWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	desc := "Nạp tiền thành công vào ví tài khoản hệ thống"
	if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.Topup, desc, &tx.ID, now); err != nil {
		return err
	}
	return nil
}

// executeDirectPurchaseWorkflow fetches the targeted subscription plan and applies it.
func (uc *PaymentWebhookUseCase) executeDirectPurchaseWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	if tx.SubscriptionPlanID == nil {
		return subscriptionerrors.ErrLinkedPlanNotFound
	}

	plan, err := uc.planRepo.GetByID(txCtx, *tx.SubscriptionPlanID)
	if err != nil {
		return subscriptionerrors.ErrLoadPlanFailed
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound
	}

	// Apply the subscription plan configuration to the user's subscription entity.
	if err := subscription.ApplySubscriptionPlan(txCtx, uc.userSubRepo, tx.UserID, plan, now); err != nil {
		return err
	}

	return nil
}
