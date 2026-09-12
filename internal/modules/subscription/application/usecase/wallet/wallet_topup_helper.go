package wallet

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/modules/subscription/application/ordercode"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/payosdescription"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
)

const walletTopUpCheckoutRetries = 3

// requestTopUpCheckoutWithRetry calls PayOS for the top-up row. When the provider
// reports a duplicate order code (231), the row has never been registered on the
// provider side, so its code is rotated in place and the call retried.
// A nil result with non-nil error signals an infrastructure failure while rotating.
func (uc *WalletUseCase) requestTopUpCheckoutWithRetry(ctx context.Context, tx *entities.DepositTransaction, returnURL, cancelURL string, expiresAt time.Time) (*payment.CheckoutSessionResult, error) {
	var result *payment.CheckoutSessionResult
	var gatewayErr error
	for attempt := 0; attempt < walletTopUpCheckoutRetries; attempt++ {
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
		if err := uc.refreshTopUpOrderCode(ctx, tx); err != nil {
			return nil, err
		}
	}
	return result, gatewayErr
}

// refreshTopUpOrderCode rotates the order code of a row still in Creating status.
func (uc *WalletUseCase) refreshTopUpOrderCode(ctx context.Context, tx *entities.DepositTransaction) error {
	nextCode := ordercode.Generate()
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if locked.Status != depositstatus.Creating {
			return subscriptionerrors.ErrDepositInitFailed()
		}
		locked.OrderCode = nextCode
		return uc.depositTxRepo.Update(txCtx, locked)
	}); err != nil {
		return err
	}
	tx.OrderCode = nextCode
	return nil
}

// applyTopUpCheckoutOutcome persists the gateway result. An exhausted duplicate
// (231) lands in CreationFailed so the reconciler never probes a foreign link.
func (uc *WalletUseCase) applyTopUpCheckoutOutcome(ctx context.Context, tx *entities.DepositTransaction, result *payment.CheckoutSessionResult, gatewayErr error) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if gatewayErr == nil && result != nil && result.Outcome == payment.OutcomeSucceeded {
			locked.Status = depositstatus.Pending
			locked.PaymentUrl = &result.CheckoutURL
		} else if result != nil && (payment.IsDuplicateOrderCode(result) || result.Outcome == payment.OutcomeKnownFailure && !result.Retryable) {
			locked.Status = depositstatus.CreationFailed
			locked.FailureReason = &result.ErrorCode
		} else {
			locked.Status = depositstatus.ReconciliationRequired
			locked.ReconciliationAttempts++
		}
		if err := uc.depositTxRepo.Update(txCtx, locked); err != nil {
			return err
		}
		return nil
	})
}
