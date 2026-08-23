package webhook

import (
	"context"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefitresolution"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/usecase/wallet"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"time"
)

func (uc *PaymentWebhookUseCase) executeWalletTopUpWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	desc := "Nạp tiền thành công vào ví tài khoản hệ thống"
	metadata := wallet.WalletStatementMetadata{}
	if err := wallet.ProcessWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.Topup, desc, &tx.ID, metadata, now); err != nil {
		return err
	}
	r := benefitresolution.WalletTopupCredited
	tx.BenefitResolution = &r

	if user, userErr := uc.userBriefContract.GetByID(txCtx, tx.UserID); userErr == nil && user != nil && user.Email != "" {
		walletEntity, _ := uc.walletRepo.GetByUserID(txCtx, tx.UserID)
		newBalance := sharedmoney.Zero
		if walletEntity != nil {
			newBalance = walletEntity.Balance
		}
		_ = uc.notifContract.Send(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateTopupSuccess, map[string]any{
			"Amount":     sharedmoney.ToFloat(tx.Amount),
			"NewBalance": sharedmoney.ToFloat(newBalance),
		}, user.Email)
	}

	return nil
}
