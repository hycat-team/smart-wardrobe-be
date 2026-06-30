package webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/subscription"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefitresolution"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/walletstatementtype"
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
	cfg                   *config.Config
	walletRepo            repositories.IUserWalletRepository
	depositTxRepo         repositories.IDepositTransactionRepository
	statementRepo         repositories.IWalletStatementRepository
	planRepo              repositories.ISubscriptionPlanRepository
	userSubRepo           repositories.IUserSubscriptionRepository
	paymentGateway        payment.IPaymentGatewayService
	uow                   shared_repos.IUnitOfWork
	paymentEventRepo      repositories.IPaymentEventRepository
	inboxRepo             repositories.IWebhookInboxRepository
	subscriptionEventRepo repositories.IUserSubscriptionEventRepository
	log                   logger.Interface
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
	paymentEventRepo repositories.IPaymentEventRepository,
	inboxRepo repositories.IWebhookInboxRepository,
	subscriptionEventRepo repositories.IUserSubscriptionEventRepository,
	log logger.Interface,
) uc_interfaces.IPaymentWebhookUseCase {
	return &PaymentWebhookUseCase{
		cfg:                   cfg,
		walletRepo:            walletRepo,
		depositTxRepo:         depositTxRepo,
		statementRepo:         statementRepo,
		planRepo:              planRepo,
		userSubRepo:           userSubRepo,
		paymentGateway:        paymentGateway,
		uow:                   uow,
		paymentEventRepo:      paymentEventRepo,
		inboxRepo:             inboxRepo,
		subscriptionEventRepo: subscriptionEventRepo,
		log:                   log,
	}
}

// ProcessWebhook verifies, validates, and applies a successful payment callback exactly once.
func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	payload, err := uc.parseVerifiedWebhookPayload(ctx, rawBody, signature)
	if err != nil {
		return err
	}
	inbox, err := uc.persistWebhookInbox(ctx, payload)
	if err != nil {
		return err
	}

	if !uc.isSuccessfulWebhook(payload) {
		uc.logIgnoredWebhook(payload)
		if inbox != nil {
			_ = uc.inboxRepo.MarkProcessed(ctx, inbox.ID, time.Now().UTC())
		}
		return nil
	}

	tx, webhookAmount, err := uc.loadAndValidateDepositTransaction(ctx, payload)
	if err != nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkInvestigation(ctx, inbox.ID, time.Now().UTC(), err.Error())
		}
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
		if inbox != nil {
			_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, time.Now().UTC(), err.Error())
		}
		uc.log.Error("Failed to process payment webhook",
			zap.Int64("order_code", payload.Data.OrderCode),
			zap.String("gateway_reference", payload.Data.Reference),
			zap.String("currency", payload.Data.Currency),
			zap.String("result", "failed"),
			zap.Error(err),
		)
		return err
	}
	if inbox != nil {
		_ = uc.inboxRepo.MarkProcessed(ctx, inbox.ID, time.Now().UTC())
	}

	uc.log.Info("Processed payment webhook successfully",
		zap.Int64("order_code", payload.Data.OrderCode),
		zap.String("gateway_reference", payload.Data.Reference),
		zap.String("currency", payload.Data.Currency),
		zap.String("result", "processed"),
	)
	return nil
}

func (uc *PaymentWebhookUseCase) CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error {
	if info == nil || info.Status != payment.ProviderPaid {
		return subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	amount := info.AmountPaid
	if amount.IsZero() {
		amount = info.Amount
	}
	payload := &WebhookPayload{Code: "00", Success: true, Data: WebhookPayloadData{OrderCode: info.OrderCode, Amount: json.Number(amount.String()), Reference: info.Reference, Currency: info.Currency, PaymentLinkId: info.PaymentLinkID, Code: "00"}}
	tx, verifiedAmount, err := uc.loadAndValidateDepositTransaction(ctx, payload)
	if err != nil {
		return err
	}
	details, err := uc.serializeWebhookDetails(payload)
	if err != nil {
		return err
	}
	return uc.processLockedWebhookTransaction(ctx, payload, tx, verifiedAmount, details)
}

// parseVerifiedWebhookPayload verifies the signature and decodes the callback body.
func (uc *PaymentWebhookUseCase) parseVerifiedWebhookPayload(ctx context.Context, rawBody []byte, signature string) (*WebhookPayload, error) {
	if err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature); err != nil {
		uc.log.Warn("Failed to verify PayOS webhook signature", zap.Error(err))
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
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		if lockedTx.Status == depositstatus.Success {
			if lockedTx.SuccessfulProviderReference == nil || *lockedTx.SuccessfulProviderReference != payload.Data.Reference {
				return subscriptionerrors.ErrProcessPaymentGatewayFailed()
			}
			return nil
		}
		if payload.Data.Reference != "" {
			existing, err := uc.paymentEventRepo.GetByReference(txCtx, "PAYOS", payload.Data.Reference)
			if err != nil {
				return err
			}
			if existing != nil {
				if existing.OrderCode != payload.Data.OrderCode || !existing.Amount.Equal(webhookAmount) || existing.PaymentLinkID != payload.Data.PaymentLinkId {
					return subscriptionerrors.ErrProcessPaymentGatewayFailed()
				}
			} else if err := uc.paymentEventRepo.Create(txCtx, &entities.ProviderPaymentEvent{Provider: "PAYOS", ProviderReference: payload.Data.Reference, EventCode: payload.Data.Code, OrderCode: payload.Data.OrderCode, PaymentLinkID: payload.Data.PaymentLinkId, Amount: webhookAmount, Currency: lockedTx.Currency}); err != nil {
				return err
			}
		}
		if err := uc.dispatchWebhookWorkflow(txCtx, lockedTx, now); err != nil {
			return err
		}
		return uc.markWebhookTransactionSuccess(txCtx, lockedTx, payload.Data.Reference, detailsStr, now)
	})
}

func (uc *PaymentWebhookUseCase) persistWebhookInbox(ctx context.Context, payload *WebhookPayload) (*entities.ProviderWebhookInbox, error) {
	canonical, err := json.Marshal(payload.Data)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	hash := hex.EncodeToString(sum[:])
	if existing, err := uc.inboxRepo.GetByHash(ctx, "PAYOS", hash); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	if payload.Data.Reference != "" {
		if existing, err := uc.inboxRepo.GetByReference(ctx, "PAYOS", payload.Data.Reference, payload.Data.Code); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	}
	amount, err := decimal.NewFromString(payload.Data.Amount.String())
	if err != nil {
		return nil, subscriptionerrors.ErrInvalidAmount()
	}
	ref, link := payload.Data.Reference, payload.Data.PaymentLinkId
	inbox := &entities.ProviderWebhookInbox{Provider: "PAYOS", ProviderReference: &ref, EventCode: payload.Data.Code, OrderCode: payload.Data.OrderCode, PaymentLinkID: &link, Amount: amount, Currency: currency.Currency(payload.Data.Currency), CanonicalPayloadHash: hash, RawPayload: entities.JSONDocument(canonical), ProcessingStatus: "RECEIVED", ReceivedAt: time.Now().UTC()}
	if err := uc.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, err
	}
	return inbox, nil
}

// markWebhookTransactionSuccess updates the deposit transaction status after a successful callback.
func (uc *PaymentWebhookUseCase) markWebhookTransactionSuccess(
	txCtx context.Context,
	lockedTx *entities.DepositTransaction,
	reference string,
	detailsStr string,
	now time.Time,
) error {
	lockedTx.Status = depositstatus.Success
	lockedTx.SuccessfulProviderReference = &reference
	lockedTx.GatewayDetails = &detailsStr
	lockedTx.BenefitAppliedAt = &now
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
	metadata := wallet.WalletStatementMetadata{}
	if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.Topup, desc, &tx.ID, metadata, now); err != nil {
		return err
	}
	r := benefitresolution.WalletTopupCredited
	tx.BenefitResolution = &r
	return nil
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

	if tx.PlanCode != nil {
		plan.Slug = *tx.PlanCode
	}
	if tx.PlanName != nil {
		plan.Name = *tx.PlanName
	}
	if tx.TierRank != nil {
		plan.TierRank = *tx.TierRank
	}
	if tx.PlanKind != nil {
		plan.PlanKind = *tx.PlanKind
	}
	plan.DurationDays = tx.PurchasedDurationDays
	free, err := uc.planRepo.GetDefaultPlan(txCtx)
	if err != nil || free == nil {
		return subscriptionerrors.ErrDefaultPlanLoadFailed()
	}
	sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, tx.UserID)
	if err != nil {
		return err
	}
	if sub == nil {
		sub = &entities.UserSubscription{UserID: tx.UserID, SubscriptionPlanID: free.ID, SubscriptionPlan: free, CurrentPlanCode: free.Slug, CurrentTierRank: free.TierRank, CurrentPlanKind: plankind.DefaultFree, CurrentBenefitSnapshot: entities.JSONDocument(`{}`), StartedAt: now}
		if err := uc.userSubRepo.ProvisionDefault(txCtx, sub); err != nil {
			return err
		}
		sub, err = uc.userSubRepo.GetByUserIDWithLock(txCtx, tx.UserID)
		if err != nil {
			return err
		}
	}
	if sub.ExpiresAt != nil && !sub.ExpiresAt.After(now) && sub.FallbackPlanID != nil {
		subEvent, err := sub.RestoreFallback("", now)
		if err != nil {
			return err
		}
		if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
			return err
		}
		if subEvent != nil {
			key := fmt.Sprintf("LIFETIME_FALLBACK_RESTORED:%s:%s", sub.UserID, sub.StartedAt.UTC().Format(time.RFC3339))
			subEvent.EventKey = key
			if existing, err := uc.subscriptionEventRepo.GetByKey(txCtx, key); err != nil {
				return err
			} else if existing == nil {
				if err := uc.subscriptionEventRepo.Create(txCtx, subEvent); err != nil {
					return err
				}
			}
		}
	}
	effective, err := subscription.ResolveEffectiveSubscription(sub, free, now)
	if err != nil {
		return err
	}
	transition := subscription.EvaluateSubscriptionTransition(subscription.PaymentCompletion, effective, subscription.PaymentSnapshot{PlanCode: plan.Slug, TierRank: plan.TierRank, PlanKind: plan.PlanKind, DurationDays: plan.DurationDays, BenefitSnapshot: tx.BenefitSnapshot})
	if transition == subscription.CreditWalletLowerTier || transition == subscription.CreditWalletSameLifetime {
		statementType, resolution := walletstatementtype.LowerTierPaymentCredit, benefitresolution.LowerTierPaymentCreditedToWallet
		if transition == subscription.CreditWalletSameLifetime {
			statementType, resolution = walletstatementtype.SameLifetimePaymentCredit, benefitresolution.SameLifetimePaymentCreditedToWallet
		}
		metadata := wallet.WalletStatementMetadata{}
		if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, statementType, "Khoản thanh toán được hoàn vào ví do gói hiện tại có quyền lợi cao hơn hoặc tương đương", &tx.ID, metadata, now); err != nil {
			return err
		}
		tx.BenefitResolution = &resolution
		credit := tx.Amount
		tx.WalletCreditAmount = &credit
		return nil
	}
	before := sub.Version
	resolution, subEvent, err := subscription.ApplyTransition(sub, plan, tx.BenefitSnapshot, transition, now)
	if err != nil {
		return err
	}
	sub.LastDepositTransactionID = &tx.ID
	if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
		return err
	}
	eventKey := fmt.Sprintf("PAYMENT_BENEFIT:%s", tx.ID)
	if subEvent != nil {
		subEvent.EventKey = eventKey
		subEvent.SourceDepositTransactionID = &tx.ID
		if existing, err := uc.subscriptionEventRepo.GetByKey(txCtx, eventKey); err != nil {
			return err
		} else if existing == nil {
			if err := uc.subscriptionEventRepo.Create(txCtx, subEvent); err != nil {
				return err
			}
		}
	}
	after := sub.Version
	tx.BenefitResolution = &resolution
	if subEvent != nil {
		tx.BenefitResultSnapshot = subEvent.Metadata
	}
	tx.SubscriptionVersionBefore = &before
	tx.SubscriptionVersionAfter = &after
	return nil
}
