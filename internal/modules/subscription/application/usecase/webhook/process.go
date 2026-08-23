package webhook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/pkg/utils/timeutils"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

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

func (uc *PaymentWebhookUseCase) serializeWebhookDetails(payload *WebhookPayload) (string, error) {
	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return "", subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	return string(rawBytes), nil
}

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
