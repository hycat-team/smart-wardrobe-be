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

// WebhookPayload represents the payment gateway callback body used by the webhook processor.
type WebhookPayload struct {
	Code      string             `json:"code"`
	Desc      string             `json:"desc"`
	Success   bool               `json:"success"`
	Data      WebhookPayloadData `json:"data"`
	Signature string             `json:"signature"`
}

// WebhookPayloadData stores the transaction-specific fields of the gateway callback.
type WebhookPayloadData struct {
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
}

// PaymentWebhookUseCase handles verified payment gateway callbacks and applies their side effects.
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

// NewPaymentWebhookUseCase builds the payment webhook processor with all required collaborators.
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

// ProcessWebhook verifies, validates, and applies a successful payment callback exactly once.
func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	payload, err := uc.parseVerifiedWebhookPayload(ctx, rawBody, signature)
	if err != nil {
		return err
	}

	if !uc.isSuccessfulWebhook(payload) {
		uc.logIgnoredWebhook(payload)
		return nil
	}

	tx, webhookAmount, err := uc.loadAndValidateDepositTransaction(ctx, payload)
	if err != nil {
		return err
	}

	if tx.Status == depositstatus.Success {
		return nil
	}

	detailsStr, err := uc.serializeWebhookDetails(payload)
	if err != nil {
		return err
	}

	if err := uc.processLockedWebhookTransaction(ctx, payload, tx, webhookAmount, detailsStr); err != nil {
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

// parseVerifiedWebhookPayload verifies the signature and decodes the callback body.
func (uc *PaymentWebhookUseCase) parseVerifiedWebhookPayload(ctx context.Context, rawBody []byte, signature string) (*WebhookPayload, error) {
	if _, err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature); err != nil {
		return nil, subscriptionerrors.ErrVerifySignatureFailed()
	}

	var payload WebhookPayload
	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, subscriptionerrors.ErrWebhookPayloadMalformed()
	}

	return &payload, nil
}

// isSuccessfulWebhook reports whether the gateway marked the callback as successful.
func (uc *PaymentWebhookUseCase) isSuccessfulWebhook(payload *WebhookPayload) bool {
	return payload.Code == "00" && payload.Data.Code == "00"
}

// logIgnoredWebhook records a non-success callback without applying any business side effects.
func (uc *PaymentWebhookUseCase) logIgnoredWebhook(payload *WebhookPayload) {
	uc.log.Warn("Ignored unsuccessful payment webhook",
		zap.Int64("order_code", payload.Data.OrderCode),
		zap.String("gateway_code", payload.Code),
		zap.String("gateway_data_code", payload.Data.Code),
		zap.String("result", "ignored_non_success"),
	)
}

// loadAndValidateDepositTransaction loads the pending deposit transaction and validates the callback amount.
func (uc *PaymentWebhookUseCase) loadAndValidateDepositTransaction(
	ctx context.Context,
	payload *WebhookPayload,
) (*entities.DepositTransaction, decimal.Decimal, error) {
	webhookAmount, err := decimal.NewFromString(payload.Data.Amount.String())
	if err != nil {
		return nil, decimal.Zero, subscriptionerrors.ErrInvalidAmount()
	}
	if err := sharedmoney.ValidateSupportedCurrencyText(payload.Data.Currency); err != nil {
		return nil, decimal.Zero, err
	}

	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, payload.Data.OrderCode)
	if err != nil {
		return nil, decimal.Zero, subscriptionerrors.ErrQueryTransactionFailed()
	}
	if tx == nil {
		return nil, decimal.Zero, subscriptionerrors.ErrTransactionNotFound(payload.Data.OrderCode)
	}
	if err := sharedmoney.ValidateSupportedCurrency(tx.Currency); err != nil {
		return nil, decimal.Zero, err
	}
	if !webhookAmount.Equal(tx.Amount) {
		return nil, decimal.Zero, subscriptionerrors.ErrPaymentAmountMismatch()
	}

	return tx, webhookAmount, nil
}

// serializeWebhookDetails stores the gateway payload for audit and replay analysis.
func (uc *PaymentWebhookUseCase) serializeWebhookDetails(payload *WebhookPayload) (string, error) {
	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return "", subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	return string(rawBytes), nil
}

// processLockedWebhookTransaction rechecks idempotency under lock and dispatches the right payment workflow.
func (uc *PaymentWebhookUseCase) processLockedWebhookTransaction(
	ctx context.Context,
	payload *WebhookPayload,
	tx *entities.DepositTransaction,
	webhookAmount decimal.Decimal,
	detailsStr string,
) error {
	_ = webhookAmount
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		lockedTx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, payload.Data.OrderCode)
		if err != nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if lockedTx == nil {
			return subscriptionerrors.ErrTransactionNotFound(payload.Data.OrderCode)
		}
		if lockedTx.Status == depositstatus.Success {
			return nil
		}

		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		if err := uc.markWebhookTransactionSuccess(txCtx, lockedTx, payload.Data.Reference, detailsStr); err != nil {
			return err
		}

		return uc.dispatchWebhookWorkflow(txCtx, lockedTx, now)
	})
}

// markWebhookTransactionSuccess updates the deposit transaction status after a successful callback.
func (uc *PaymentWebhookUseCase) markWebhookTransactionSuccess(
	txCtx context.Context,
	lockedTx *entities.DepositTransaction,
	reference string,
	detailsStr string,
) error {
	lockedTx.Status = depositstatus.Success
	lockedTx.GatewayReference = &reference
	lockedTx.GatewayDetails = &detailsStr
	if err := uc.depositTxRepo.Update(txCtx, lockedTx); err != nil {
		return subscriptionerrors.ErrCompletePaymentRecordFailed()
	}
	return nil
}

// dispatchWebhookWorkflow routes the locked transaction to the proper wallet or subscription workflow.
func (uc *PaymentWebhookUseCase) dispatchWebhookWorkflow(txCtx context.Context, lockedTx *entities.DepositTransaction, now time.Time) error {
	switch lockedTx.TransactionType {
	case deposittransactiontype.WalletTopup:
		return uc.executeWalletTopUpWorkflow(txCtx, lockedTx, now)
	case deposittransactiontype.DirectPurchase:
		return uc.executeDirectPurchaseWorkflow(txCtx, lockedTx, now)
	default:
		return nil
	}
}

// executeWalletTopUpWorkflow credits the user's wallet after a successful top-up payment.
func (uc *PaymentWebhookUseCase) executeWalletTopUpWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	desc := "Nạp tiền thành công vào ví tài khoản hệ thống"
	return wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.Topup, desc, &tx.ID, now)
}

// executeDirectPurchaseWorkflow fetches the targeted subscription plan and applies it.
func (uc *PaymentWebhookUseCase) executeDirectPurchaseWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	if tx.SubscriptionPlanID == nil {
		return subscriptionerrors.ErrLinkedPlanNotFound()
	}

	plan, err := uc.planRepo.GetByID(txCtx, *tx.SubscriptionPlanID)
	if err != nil {
		return subscriptionerrors.ErrLoadPlanFailed()
	}
	if plan == nil {
		return subscriptionerrors.ErrRequestedPlanNotFound()
	}

	return subscription.ApplySubscriptionPlan(txCtx, uc.userSubRepo, tx.UserID, plan, now)
}
