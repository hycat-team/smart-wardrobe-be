package reconciliation

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	errCodeProviderLookup   = "PROVIDER_LOOKUP_ERROR"
	errCodeCompletion       = "COMPLETION_ERROR"
	errCodeCancelUnknown    = "CANCEL_UNKNOWN"
	errCodeCreateTimeout    = "CREATE_TIMEOUT"
	errCodeCreateUnknown    = "CREATE_UNKNOWN"
	errCodePendingRetry     = "PENDING_RETRY"
	errCodeUnknownStatus    = "UNKNOWN_PROVIDER_STATUS"
	reasonMaxPolicyExceeded = "MAX_RECONCILIATION_POLICY_EXCEEDED"

	cancelReasonExpired = "Payment link expired"
)

type PaymentReconciliationUseCase struct {
	repo       repositories.IDepositTransactionRepository
	gateway    payment.IPaymentGatewayService
	completion usecase_interfaces.IPaymentWebhookUseCase
	uow        shared_repos.IUnitOfWork
	log        logger.Interface
	cfg        *config.Config
}

func NewPaymentReconciliationUseCase(
	repo repositories.IDepositTransactionRepository,
	gateway payment.IPaymentGatewayService,
	completion usecase_interfaces.IPaymentWebhookUseCase,
	uow shared_repos.IUnitOfWork,
	log logger.Interface,
	cfg *config.Config,
) usecase_interfaces.IPaymentReconciliationUseCase {
	return &PaymentReconciliationUseCase{
		repo:       repo,
		gateway:    gateway,
		completion: completion,
		uow:        uow,
		log:        log,
		cfg:        cfg,
	}
}

func (uc *PaymentReconciliationUseCase) Reconcile(ctx context.Context, run *workerlog.Run) error {
	rows, err := uc.repo.ClaimReconciliationCandidates(ctx, uc.cfg.PayOS.ReconciliationBatchSize, time.Duration(uc.cfg.PayOS.ReconciliationLeaseSeconds)*time.Second)
	if err != nil {
		run.ChildError(uc.log, "Failed to claim payment reconciliation candidates", zap.Error(err))
		return err
	}
	run.AddSummaryFields(
		zap.Int("pendingRetryCount", 0),
		zap.Int("expiredCount", 0),
		zap.Int("cancelledCount", 0),
		zap.Int("investigationRequiredCount", 0),
		zap.Int("recoveredToPendingCount", 0),
		zap.Int("alreadyHandledCount", 0),
		zap.Int("zombieSkippedCount", 0),
	)
	for _, row := range rows {
		run.AddTotal(1)
		uc.reconcile(ctx, row, run)
	}
	return nil
}

func (uc *PaymentReconciliationUseCase) reconcile(ctx context.Context, claimed *repositories.ClaimedDepositTransaction, runs ...*workerlog.Run) {
	var run *workerlog.Run
	if len(runs) > 0 {
		run = runs[0]
	}
	if run == nil {
		run = workerlog.New("payment_reconciliation", "internal")
	}
	orderCode := claimed.Transaction.OrderCode
	token := claimed.ProcessingToken
	tx := claimed.Transaction

	info, err := uc.gateway.GetPaymentLinkInfo(ctx, orderCode)
	if err != nil {
		uc.deferRetry(ctx, orderCode, token, errCodeProviderLookup, run)
		return
	}

	switch info.Status {
	case payment.ProviderPaid:
		if err := uc.completion.CompleteVerifiedPayment(ctx, info); err != nil {
			uc.deferRetry(ctx, orderCode, token, errCodeCompletion, run)
			return
		}
		run.AddSuccess(1)
	case payment.ProviderPending:
		now := time.Now().UTC()
		// If expiresAt <= now: cancel the link and mark expired
		if tx.ExpiresAt != nil && !tx.ExpiresAt.After(now) {
			if err := uc.gateway.CancelPaymentLink(ctx, orderCode, cancelReasonExpired); err != nil {
				uc.deferRetry(ctx, orderCode, token, errCodeCancelUnknown, run)
				return
			}
			uc.finish(ctx, orderCode, token, depositstatus.Expired, run)
			return
		}

		// If expiresAt > now: check claimedFromStatus
		if tx.ExpiresAt != nil && tx.ExpiresAt.After(now) {
			isRecoverable := claimed.ClaimedFromStatus == depositstatus.Creating ||
				(claimed.ClaimedFromStatus == depositstatus.ReconciliationRequired &&
					tx.LastProviderErrorCode != nil &&
					(*tx.LastProviderErrorCode == errCodeProviderLookup || *tx.LastProviderErrorCode == errCodeCreateTimeout || *tx.LastProviderErrorCode == errCodeCreateUnknown))

			if isRecoverable {
				updates := map[string]any{
					"status":                 depositstatus.Pending,
					"processing_token":       nil,
					"processing_lease_until": nil,
				}
				if info.PaymentLinkID != "" {
					updates["payment_link_id"] = info.PaymentLinkID
				}
				if info.CheckoutURL != "" {
					updates["payment_url"] = info.CheckoutURL
				}

				rowsAffected, err := uc.repo.UpdateWithToken(ctx, orderCode, token, updates)
				if err != nil {
					run.ChildError(uc.log, "Failed to recover transaction to Pending", zap.Int64("orderCode", orderCode), zap.Error(err))
				} else if rowsAffected == 0 {
					uc.handleZeroRowsAffected(ctx, orderCode, token, run)
				} else {
					run.AddSuccess(1)
					run.AddSummaryFields(zap.Int("recoveredToPendingCount", 1))
				}
				return
			}

			// If it's cancel timeout error, try cancel again
			if claimed.ClaimedFromStatus == depositstatus.ReconciliationRequired &&
				tx.LastProviderErrorCode != nil &&
				*tx.LastProviderErrorCode == errCodeCancelUnknown {
				if err := uc.gateway.CancelPaymentLink(ctx, orderCode, cancelReasonExpired); err != nil {
					uc.deferRetry(ctx, orderCode, token, errCodeCancelUnknown, run)
					return
				}
				uc.finish(ctx, orderCode, token, depositstatus.Expired, run)
				return
			}
		}

		uc.deferRetry(ctx, orderCode, token, errCodePendingRetry, run)

	case payment.ProviderCancelled:
		uc.finish(ctx, orderCode, token, depositstatus.Cancelled, run)
	default:
		uc.deferRetry(ctx, orderCode, token, errCodeUnknownStatus, run)
	}
}

func (uc *PaymentReconciliationUseCase) handleZeroRowsAffected(ctx context.Context, orderCode int64, token uuid.UUID, run *workerlog.Run) {
	row, err := uc.repo.GetByOrderCode(ctx, orderCode)
	if err != nil || row == nil {
		return
	}
	if row.Status == depositstatus.Success {
		run.ChildWarn(uc.log, "Transaction was already marked Success by Webhook. Worker gracefully exits.", zap.Int64("orderCode", orderCode))
		run.AddSkipped(1)
		return
	}
	if row.ProcessingToken == nil || *row.ProcessingToken != token {
		run.ChildWarn(uc.log, "Worker became a zombie for transaction. Another worker claimed it.", zap.Int64("orderCode", orderCode))
		run.AddSkipped(1)
		return
	}
}

func (uc *PaymentReconciliationUseCase) finish(ctx context.Context, orderCode int64, token uuid.UUID, status depositstatus.DepositStatus, run *workerlog.Run) {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":                 status,
		"processing_token":       nil,
		"processing_lease_until": nil,
	}
	if status == depositstatus.Expired {
		updates["expired_at"] = &now
	} else {
		updates["cancelled_at"] = &now
	}

	rowsAffected, err := uc.repo.UpdateWithToken(ctx, orderCode, token, updates)
	if err != nil {
		run.ChildError(uc.log, "Failed to update finished transaction", zap.Int64("orderCode", orderCode), zap.Error(err))
	} else if rowsAffected == 0 {
		uc.handleZeroRowsAffected(ctx, orderCode, token, run)
	} else {
		run.AddSuccess(1)
	}
}

func (uc *PaymentReconciliationUseCase) deferRetry(ctx context.Context, orderCode int64, token uuid.UUID, code string, run *workerlog.Run) {
	row, err := uc.repo.GetByOrderCode(ctx, orderCode)
	if err != nil || row == nil {
		return
	}
	if row.Status == depositstatus.Success {
		return
	}
	if row.ProcessingToken == nil || *row.ProcessingToken != token {
		return
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"processing_token":       nil,
		"processing_lease_until": nil,
	}

	if row.ReconciliationAttempts >= uc.cfg.PayOS.ReconciliationMaxAttempts || now.Sub(row.CreatedAt) >= time.Duration(uc.cfg.PayOS.ReconciliationMaxAgeHours)*time.Hour {
		reason := reasonMaxPolicyExceeded
		updates["status"] = depositstatus.InvestigationRequired
		updates["failure_reason"] = &reason
		updates["next_reconciliation_at"] = nil
	} else {
		next := now.Add(time.Duration(1<<min(row.ReconciliationAttempts, 6)) * time.Minute)
		updates["status"] = depositstatus.ReconciliationRequired
		updates["reconciliation_attempts"] = row.ReconciliationAttempts + 1
		updates["last_provider_error_code"] = &code
		updates["last_provider_error_at"] = &now
		updates["next_reconciliation_at"] = &next
	}

	rowsAffected, err := uc.repo.UpdateWithToken(ctx, orderCode, token, updates)
	if err != nil {
		run.ChildError(uc.log, "Failed to defer retry for transaction", zap.Int64("orderCode", orderCode), zap.String("errorCode", code), zap.Error(err))
	} else if rowsAffected == 0 {
		uc.handleZeroRowsAffected(ctx, orderCode, token, run)
	} else {
		run.ChildWarn(uc.log, "Transaction deferred for retry", zap.Int64("orderCode", orderCode), zap.String("errorCode", code))
		run.AddRetry(1)
	}
}
