package webhook

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/walletstatementtype"

	"go.uber.org/zap"
)

func (uc *PaymentWebhookUseCase) AutoRefund(ctx context.Context, orderCode int64, reason string) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		tx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, orderCode)
		if err != nil {
			return err
		}
		if tx == nil {
			return subscriptionerrors.ErrTransactionNotFound(orderCode)
		}
		if tx.Status == depositstatus.Refunded {
			return nil
		}
		now := time.Now().UTC()
		metadata := wallet.WalletStatementMetadata{}
		if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.AutoRefund, "Tự động hoàn tiền: "+reason, &tx.ID, metadata, now); err != nil {
			return err
		}
		tx.Status = depositstatus.Refunded
		tx.RefundedAt = &now
		tx.RefundAmount = &tx.Amount
		tx.RefundReason = &reason
		tx.BenefitResolution = nil
		if err := uc.depositTxRepo.Update(txCtx, tx); err != nil {
			return err
		}
		uc.log.Info("Auto-refund completed",
			zap.Int64("orderCode", orderCode),
			zap.String("reason", reason),
			zap.String("userId", tx.UserID.String()),
		)
		return nil
	})
}
