package webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/webhookprocessingstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/shared/domain/money"
	"smart-wardrobe-be/shared/pkg/utils/timeutils"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

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

func (uc *PaymentWebhookUseCase) isSuccessfulWebhook(payload *WebhookPayload) bool {
	return payload.Code == "00" && payload.Data.Code == "00"
}

func (uc *PaymentWebhookUseCase) logIgnoredWebhook(payload *WebhookPayload) {
	uc.log.Warn("Ignored unsuccessful payment webhook",
		zap.Int64("order_code", payload.Data.OrderCode),
		zap.String("gateway_code", payload.Code),
		zap.String("gateway_data_code", payload.Data.Code),
		zap.String("result", "ignored_non_success"),
	)
}

// resolveUnsuccessfulWebhook handles a webhook that does not report a successful
// bank transaction. The payOS webhook payload carries no payment-link status
// (only transaction fields), so the provider is queried and the authoritative
// status decides the transition. Nothing is inferred from the payload itself.
//
// A provider lookup failure is returned to the caller so payOS retries; the
// reconciliation worker still covers the row independently.
func (uc *PaymentWebhookUseCase) resolveUnsuccessfulWebhook(
	ctx context.Context,
	payload *WebhookPayload,
	inbox *entities.ProviderWebhookInbox,
) error {
	uc.logIgnoredWebhook(payload)
	now := time.Now().UTC()
	ack := func() error {
		if inbox != nil {
			_ = uc.inboxRepo.MarkProcessed(ctx, inbox.ID, now)
		}
		return nil
	}

	if payload.Data.OrderCode == 0 {
		return ack()
	}

	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, payload.Data.OrderCode)
	if err != nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, now, "transaction lookup failed")
		}
		return subscriptionerrors.ErrQueryTransactionFailed()
	}
	if tx == nil || isTerminalDepositStatus(tx.Status) {
		return ack()
	}

	info, err := uc.paymentGateway.GetPaymentLinkInfo(ctx, payload.Data.OrderCode)
	if err != nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, now, "provider lookup failed")
		}
		return err
	}
	if info == nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkInvestigation(ctx, inbox.ID, now, "provider returned no payment link info")
		}
		return nil
	}

	switch info.Status {
	case payment.ProviderPaid:
		// A successful bank transaction may still arrive with a non-"00"
		// webhook code (provider retry ordering); the provider is authoritative.
		if err := uc.CompleteVerifiedPayment(ctx, info); err != nil {
			if inbox != nil {
				_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, now, "payment completion failed")
			}
			return err
		}
	case payment.ProviderCancelled:
		if err := uc.MarkCancelled(ctx, info); err != nil {
			if inbox != nil {
				_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, now, "cancellation persistence failed")
			}
			return err
		}
	default:
		// PENDING/PROCESSING/EXPIRED/UNDERPAID/FAILED/UNKNOWN: reconciliation owns
		// these transitions. Record the signal and acknowledge so payOS does not
		// retry a payload the webhook path cannot resolve.
		message := "unsuccessful webhook; provider status " + string(info.Status)
		if inbox != nil {
			_ = uc.inboxRepo.MarkInvestigation(ctx, inbox.ID, now, message)
		}
		return nil
	}

	return ack()
}

// isTerminalDepositStatus reports states that a late webhook must not reopen.
func isTerminalDepositStatus(status depositstatus.DepositStatus) bool {
	switch status {
	case depositstatus.Success, depositstatus.FailedLegacy, depositstatus.CreationFailed,
		depositstatus.Cancelled, depositstatus.Expired, depositstatus.Refunded:
		return true
	default:
		return false
	}
}

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
	// An unknown orderCode is acknowledged, not failed. The payload is already
	// persisted in the inbox, and the reconciliation worker recovers genuine
	// races by polling the provider for pending transactions. Returning 404
	// would make the provider retry forever and fail webhook URL confirmation.
	if tx == nil {
		return nil, decimal.Zero, nil
	}
	if err := sharedmoney.ValidateSupportedCurrency(tx.Currency); err != nil {
		return nil, decimal.Zero, err
	}
	if !webhookAmount.Equal(tx.Amount) {
		return nil, decimal.Zero, subscriptionerrors.ErrPaymentAmountMismatch()
	}

	return tx, webhookAmount, nil
}

func (uc *PaymentWebhookUseCase) serializeWebhookDetails(payload *WebhookPayload) (string, error) {
	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return "", subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	return string(rawBytes), nil
}

// processLockedWebhookTransaction applies a verified payment inside a unit of
// work. It re-reads the transaction under a row lock (the caller's read is only
// a pre-check), so it takes no transaction argument.
func (uc *PaymentWebhookUseCase) processLockedWebhookTransaction(
	ctx context.Context,
	payload *WebhookPayload,
	webhookAmount decimal.Decimal,
	detailsStr string,
) error {
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
	amount := decimal.Zero
	if rawAmount := strings.TrimSpace(payload.Data.Amount.String()); rawAmount != "" {
		parsed, err := decimal.NewFromString(rawAmount)
		if err != nil {
			return nil, subscriptionerrors.ErrInvalidAmount()
		}
		amount = parsed
	}
	// A non-payment webhook (provider test sample, cancellation notice) may omit
	// amount/currency. Persist what we have instead of rejecting the request,
	// otherwise payOS retries a payload we will never accept.
	currencyText := strings.TrimSpace(payload.Data.Currency)
	if currencyText == "" {
		currencyText = string(currency.VND)
	}
	ref, link := payload.Data.Reference, payload.Data.PaymentLinkId
	inbox := &entities.ProviderWebhookInbox{Provider: "PAYOS", ProviderReference: &ref, EventCode: payload.Data.Code, OrderCode: payload.Data.OrderCode, PaymentLinkID: &link, Amount: amount, Currency: currency.Currency(currencyText), CanonicalPayloadHash: hash, RawPayload: entities.JSONDocument(canonical), ProcessingStatus: string(webhookprocessingstatus.Received), ReceivedAt: time.Now().UTC()}
	if err := uc.inboxRepo.Create(ctx, inbox); err != nil {
		return nil, err
	}
	return inbox, nil
}

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
