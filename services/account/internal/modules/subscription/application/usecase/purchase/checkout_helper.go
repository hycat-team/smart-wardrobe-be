package purchase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/ordercode"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/payosdescription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

const directPurchaseCheckoutRetries = 3

// linkReuseMinTTL is the minimum remaining lifetime for a stored payment link to
// be handed out again. Links closer to expiry are rotated so the user always pays
// with a fresh window instead of racing PayOS expiry on a weak connection.
const linkReuseMinTTL = 5 * time.Minute

const (
	cancelReasonSuperseded = "Superseded by new purchase link"
	cancelReasonStale      = "Payment link expired"
)

// isReusableDirectPurchaseLink reports whether the stored row carries a live,
// comfortably-valid payment link that can be returned as-is.
func isReusableDirectPurchaseLink(tx *entities.DepositTransaction, reuseUntil time.Time) bool {
	return tx != nil &&
		tx.Status == depositstatus.Pending &&
		tx.PaymentUrl != nil &&
		tx.ExpiresAt != nil && tx.ExpiresAt.After(reuseUntil)
}

// isExpiredDirectPurchaseLink reports whether the stored row is past its usable
// lifetime and must be closed before a new transaction is created.
func isExpiredDirectPurchaseLink(tx *entities.DepositTransaction, now time.Time) bool {
	return tx != nil && (tx.ExpiresAt == nil || !tx.ExpiresAt.After(now))
}

func mapDepositTransactionToPaymentLinkDTO(tx *entities.DepositTransaction) *dto.PaymentLinkDTO {
	url := ""
	if tx.PaymentUrl != nil {
		url = *tx.PaymentUrl
	}
	return &dto.PaymentLinkDTO{
		PaymentUrl:           url,
		OrderCode:            tx.OrderCode,
		PaymentStatus:        tx.Status,
		ExpiresAt:            tx.ExpiresAt,
		NextReconciliationAt: tx.NextReconciliationAt,
	}
}

// requestDirectPurchaseCheckoutWithRetry calls PayOS for the purchase row. When the
// provider reports a duplicate order code (231), the row has never been registered
// on the provider side, so its code is rotated in place and the call retried.
// A nil result with non-nil error signals an infrastructure failure while rotating.
func (uc *SubscriptionPurchaseUseCase) requestDirectPurchaseCheckoutWithRetry(ctx context.Context, tx *entities.DepositTransaction, returnURL, cancelURL string, expiresAt time.Time) (*payment.CheckoutSessionResult, error) {
	var result *payment.CheckoutSessionResult
	var gatewayErr error
	for attempt := 0; attempt < directPurchaseCheckoutRetries; attempt++ {
		result, gatewayErr = uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{
			OrderCode:   tx.OrderCode,
			Amount:      tx.Amount,
			Description: payosdescription.FormatTransactionDescription(tx.OrderCode),
			ReturnUrl:   returnURL,
			CancelUrl:   cancelURL,
			ExpiresAt:   expiresAt,
		})
		if gatewayErr == nil || !payment.IsDuplicateOrderCode(result) {
			return result, gatewayErr
		}
		if err := uc.refreshDirectPurchaseOrderCode(ctx, tx); err != nil {
			return nil, err
		}
	}
	return result, gatewayErr
}

// refreshDirectPurchaseOrderCode rotates the order code of a row still in Creating status.
func (uc *SubscriptionPurchaseUseCase) refreshDirectPurchaseOrderCode(ctx context.Context, tx *entities.DepositTransaction) error {
	nextCode := ordercode.Generate()
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if locked.Status != depositstatus.Creating {
			return subscriptionerrors.ErrDirectPurchaseCreateFailed()
		}
		locked.OrderCode = nextCode
		return uc.depositTxRepo.Update(txCtx, locked)
	}); err != nil {
		return err
	}
	tx.OrderCode = nextCode
	return nil
}

// applyDirectPurchaseCheckoutOutcome persists the gateway result. An exhausted
// duplicate (231) lands in CreationFailed so the reconciler never probes a foreign link.
func (uc *SubscriptionPurchaseUseCase) applyDirectPurchaseCheckoutOutcome(ctx context.Context, tx *entities.DepositTransaction, result *payment.CheckoutSessionResult, gatewayErr error, now time.Time) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
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
		} else if result != nil && (payment.IsDuplicateOrderCode(result) || result.Outcome == payment.OutcomeKnownFailure && !result.Retryable) {
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
}
