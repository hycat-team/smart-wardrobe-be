package resolution

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/depositstatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

type PaymentResolutionUseCase struct {
	repo       repositories.IDepositTransactionRepository
	completion uc_interfaces.IPaymentWebhookUseCase
	uow        shared_repos.IUnitOfWork
}

func NewPaymentResolutionUseCase(repo repositories.IDepositTransactionRepository, completion uc_interfaces.IPaymentWebhookUseCase, uow shared_repos.IUnitOfWork) uc_interfaces.IPaymentResolutionUseCase {
	return &PaymentResolutionUseCase{repo: repo, completion: completion, uow: uow}
}

func (uc *PaymentResolutionUseCase) RetryInvestigation(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	return uc.transition(ctx, input, depositstatus.ReconciliationRequired)
}

func (uc *PaymentResolutionUseCase) ResolveAsPaid(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	if input.Evidence == nil || input.Evidence.Reference == "" || input.Evidence.PaymentLinkID == "" {
		return subscriptionerrors.ErrInvalidManualPaymentResolution()
	}
	return uc.completion.CompleteVerifiedPayment(ctx, input.Evidence)
}

func (uc *PaymentResolutionUseCase) ResolveAsCancelled(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	return uc.transition(ctx, input, depositstatus.Cancelled)
}

func (uc *PaymentResolutionUseCase) ResolveAsCreationFailed(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	if input.ExpectedStatus != depositstatus.InvestigationRequired && input.ExpectedStatus != depositstatus.Creating && input.ExpectedStatus != depositstatus.ReconciliationRequired {
		return subscriptionerrors.ErrInvalidManualPaymentResolution()
	}
	return uc.transition(ctx, input, depositstatus.CreationFailed)
}

func (uc *PaymentResolutionUseCase) transition(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput, target depositstatus.DepositStatus) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		row, err := uc.repo.GetByOrderCodeWithLock(txCtx, input.OrderCode)
		if err != nil {
			return err
		}
		if row == nil || row.Status == depositstatus.Success || row.Status != input.ExpectedStatus {
			return subscriptionerrors.ErrInvalidManualPaymentResolution()
		}
		now := time.Now().UTC()
		row.Status = target
		row.FailureReason = &input.Reason
		row.ProcessingToken = nil
		row.ProcessingLeaseUntil = nil
		if target == depositstatus.ReconciliationRequired {
			row.NextReconciliationAt = &now
		} else if target == depositstatus.Cancelled {
			row.CancelledAt = &now
		}
		return uc.repo.Update(txCtx, row)
	})
}
