package webhook

import (
	"context"
	"strings"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"

	"go.uber.org/zap"
)

const (
	// failureReasonMaxLen matches deposit_transactions.failure_reason VARCHAR(100).
	failureReasonMaxLen       = 100
	defaultCancellationReason = "PAYMENT_LINK_CANCELLED"
)

// MarkCancelled records a provider-confirmed cancellation. payOS reports the
// cancellation through GetPaymentLinkInfo (the webhook payload carries no link
// status), so the provider status is authoritative here.
//
// The transition is guarded: a settled, refunded, already-cancelled or expired
// row is left untouched. The caller may hold no reconciliation lease, so the
// row is locked before the status check rather than relying on a token.
func (uc *PaymentWebhookUseCase) MarkCancelled(ctx context.Context, info *payment.PaymentLinkInfo) error {
	if info == nil {
		return subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		tx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, info.OrderCode)
		if err != nil {
			return err
		}
		if tx == nil {
			return subscriptionerrors.ErrTransactionNotFound(info.OrderCode)
		}
		switch tx.Status {
		case depositstatus.Success, depositstatus.Cancelled, depositstatus.Expired, depositstatus.Refunded:
			return nil
		}

		now := time.Now().UTC()
		cancelledAt := now
		if info.CancelledAt != nil && !info.CancelledAt.IsZero() {
			cancelledAt = info.CancelledAt.UTC()
		}
		reason := truncateFailureReason(strings.TrimSpace(info.CancellationReason))
		if reason == "" {
			reason = defaultCancellationReason
		}
		providerStatus := string(payment.ProviderCancelled)

		tx.Status = depositstatus.Cancelled
		tx.ProviderStatus = &providerStatus
		tx.CancelledAt = &cancelledAt
		tx.FailureReason = &reason
		tx.ProcessingToken = nil
		tx.ProcessingLeaseUntil = nil
		tx.NextReconciliationAt = nil
		if err := uc.depositTxRepo.Update(txCtx, tx); err != nil {
			return err
		}
		uc.log.Info("Payment link marked cancelled",
			zap.Int64("order_code", tx.OrderCode),
			zap.String("payment_link_id", info.PaymentLinkID),
			zap.String("reason", reason),
		)
		return nil
	})
}

func truncateFailureReason(reason string) string {
	runes := []rune(reason)
	if len(runes) <= failureReasonMaxLen {
		return reason
	}
	return string(runes[:failureReasonMaxLen])
}
