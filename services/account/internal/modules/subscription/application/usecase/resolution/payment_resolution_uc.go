package resolution

import (
	"context"
	"time"

	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
)

// resolutionAction names one manual reconciliation decision.
//
// The name is not decoration: it is the KEY into `statusesForAction` below, so a
// new action cannot be added without answering "which statuses may it start
// from" at the same time.
type resolutionAction string

const (
	actionResolvePaid           resolutionAction = "resolve-paid"
	actionResolveCancelled      resolutionAction = "resolve-cancelled"
	actionResolveCreationFailed resolutionAction = "resolve-creation-failed"
	actionRetryInvestigation    resolutionAction = "retry-investigation"
)

// statusesForAction pins the statuses each manual action may START from.
//
// WHY THIS EXISTS. The guard used to be "not success", which enabled buttons (and
// accepted requests) that make no sense and, in one case, are destructive:
//
//   - `refunded` already credited the wallet back; marking it paid a second time
//     would grant the entitlement again while the refund stays. Forbidden.
//   - `creation_failed` never produced a payment link, so no money can have
//     arrived through one. Forbidden.
//   - `success` is done, and an `unknown` status comes from hand-edited data this
//     API must not act on. Forbidden.
//   - `cancelled`/`expired`/`failed_legacy` still allow `resolve-paid` on purpose:
//     a customer who cancels the link and then transfers anyway is exactly the
//     case manual reconciliation exists for, and no automatic path reopens a
//     terminal transaction (see webhook.isTerminalDepositStatus).
//
// The frontend mirrors this table (features/admin/utils/transactions.ts); this one
// is the authority, because curl does not go through a disabled button.
var statusesForAction = map[resolutionAction][]depositstatus.DepositStatus{
	actionResolvePaid: {
		depositstatus.Pending,
		depositstatus.Creating,
		depositstatus.Reconciling,
		depositstatus.ReconciliationRequired,
		depositstatus.InvestigationRequired,
		depositstatus.Cancelled,
		depositstatus.Expired,
		depositstatus.FailedLegacy,
	},
	actionResolveCancelled: {
		depositstatus.Pending,
		depositstatus.Creating,
		depositstatus.Reconciling,
		depositstatus.ReconciliationRequired,
		depositstatus.InvestigationRequired,
	},
	actionRetryInvestigation: {
		depositstatus.Pending,
		depositstatus.Creating,
		depositstatus.Reconciling,
		depositstatus.ReconciliationRequired,
		depositstatus.InvestigationRequired,
	},
	actionResolveCreationFailed: {
		depositstatus.Creating,
		depositstatus.ReconciliationRequired,
		depositstatus.InvestigationRequired,
	},
}

// allowsStatus reports whether `action` may start from `status`.
func allowsStatus(action resolutionAction, status depositstatus.DepositStatus) bool {
	for _, allowed := range statusesForAction[action] {
		if allowed == status {
			return true
		}
	}
	return false
}

type PaymentResolutionUseCase struct {
	repo       repositories.IDepositTransactionRepository
	completion uc_interfaces.IPaymentWebhookUseCase
	uow        shared_repos.IUnitOfWork
}

func NewPaymentResolutionUseCase(repo repositories.IDepositTransactionRepository, completion uc_interfaces.IPaymentWebhookUseCase, uow shared_repos.IUnitOfWork) uc_interfaces.IPaymentResolutionUseCase {
	return &PaymentResolutionUseCase{repo: repo, completion: completion, uow: uow}
}

func (uc *PaymentResolutionUseCase) RetryInvestigation(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	return uc.transition(ctx, actionRetryInvestigation, input, depositstatus.ReconciliationRequired)
}

func (uc *PaymentResolutionUseCase) ResolveAsPaid(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	if input.Evidence == nil || input.Evidence.Reference == "" || input.Evidence.PaymentLinkID == "" {
		return subscriptionerrors.ErrInvalidManualPaymentResolution()
	}
	if !allowsStatus(actionResolvePaid, input.ExpectedStatus) {
		return subscriptionerrors.ErrInvalidManualPaymentResolution()
	}
	// The administrator is ASSERTING that the money arrived. The completion path
	// is the webhook's, and it refuses anything that is not a PAID link — while
	// this caller only has typed evidence, never a provider status. Leaving it
	// unset made every resolve-paid answer 500 "Không thể xử lý thông tin cổng
	// thanh toán", because the zero value of ProviderPaymentStatus is "".
	input.Evidence.Status = payment.ProviderPaid
	return uc.completion.CompleteVerifiedPayment(ctx, input.Evidence)
}

func (uc *PaymentResolutionUseCase) ResolveAsCancelled(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	return uc.transition(ctx, actionResolveCancelled, input, depositstatus.Cancelled)
}

func (uc *PaymentResolutionUseCase) ResolveAsCreationFailed(ctx context.Context, input uc_interfaces.ManualPaymentResolutionInput) error {
	return uc.transition(ctx, actionResolveCreationFailed, input, depositstatus.CreationFailed)
}

// transition applies one status change under a row lock.
//
// The status is checked TWICE, against the same table, for two different reasons:
//
//   - before the unit of work, against `input.ExpectedStatus`, so an obviously
//     invalid request answers 400 without opening a transaction;
//   - inside the lock, against the freshly read `row.Status`, because
//     `ExpectedStatus` was read by the handler before this request started. A
//     webhook or another administrator may have moved the row since, and the row
//     under lock is the only thing that is true.
func (uc *PaymentResolutionUseCase) transition(
	ctx context.Context,
	action resolutionAction,
	input uc_interfaces.ManualPaymentResolutionInput,
	target depositstatus.DepositStatus,
) error {
	if !allowsStatus(action, input.ExpectedStatus) {
		return subscriptionerrors.ErrInvalidManualPaymentResolution()
	}
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		row, err := uc.repo.GetByOrderCodeWithLock(txCtx, input.OrderCode)
		if err != nil {
			return err
		}
		if row == nil || !allowsStatus(action, row.Status) || row.Status != input.ExpectedStatus {
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
