package worker

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type IPaymentReconciliationWorker interface {
	Start()
	Stop()
}

type PaymentReconciliationWorker struct {
	repo       repositories.IDepositTransactionRepository
	gateway    payment.IPaymentGatewayService
	completion usecase_interfaces.IPaymentWebhookUseCase
	uow        shared_repos.IUnitOfWork
	cron       *cron.Cron
	log        logger.Interface
	cfg        *config.Config
}

func NewPaymentReconciliationWorker(repo repositories.IDepositTransactionRepository, gateway payment.IPaymentGatewayService, completion usecase_interfaces.IPaymentWebhookUseCase, uow shared_repos.IUnitOfWork, log logger.Interface, cfg *config.Config) IPaymentReconciliationWorker {
	return &PaymentReconciliationWorker{repo: repo, gateway: gateway, completion: completion, uow: uow, cron: cron.New(cron.WithSeconds()), log: log, cfg: cfg}
}
func (w *PaymentReconciliationWorker) Start() {
	_, err := w.cron.AddFunc(w.cfg.PayOS.ReconciliationCron, w.run)
	if err != nil {
		w.log.Error("Failed to register payment reconciliation worker", zap.Error(err))
		return
	}
	w.cron.Start()
}
func (w *PaymentReconciliationWorker) Stop() {
	if w.cron != nil {
		w.cron.Stop()
	}
}
func (w *PaymentReconciliationWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rows, err := w.repo.ClaimReconciliationCandidates(ctx, w.cfg.PayOS.ReconciliationBatchSize, time.Duration(w.cfg.PayOS.ReconciliationLeaseSeconds)*time.Second)
	if err != nil {
		w.log.Error("Failed to claim payment reconciliation candidates", zap.Error(err))
		return
	}
	for _, row := range rows {
		w.reconcile(ctx, row)
	}
}

func (w *PaymentReconciliationWorker) reconcile(ctx context.Context, claimed *repositories.ClaimedDepositTransaction) {
	orderCode := claimed.Transaction.OrderCode
	token := claimed.ProcessingToken
	tx := claimed.Transaction

	info, err := w.gateway.GetPaymentLinkInfo(ctx, orderCode)
	if err != nil {
		w.deferRetry(ctx, orderCode, token, "PROVIDER_LOOKUP_ERROR")
		return
	}

	switch info.Status {
	case payment.ProviderPaid:
		if err := w.completion.CompleteVerifiedPayment(ctx, info); err != nil {
			w.deferRetry(ctx, orderCode, token, "COMPLETION_ERROR")
		}
	case payment.ProviderPending:
		now := time.Now().UTC()
		// If expiresAt <= now: cancel the link and mark expired
		if tx.ExpiresAt != nil && !tx.ExpiresAt.After(now) {
			if err := w.gateway.CancelPaymentLink(ctx, orderCode, "Payment link expired"); err != nil {
				w.deferRetry(ctx, orderCode, token, "CANCEL_UNKNOWN")
				return
			}
			w.finish(ctx, orderCode, token, depositstatus.Expired)
			return
		}

		// If expiresAt > now: check claimedFromStatus
		if tx.ExpiresAt != nil && tx.ExpiresAt.After(now) {
			isRecoverable := claimed.ClaimedFromStatus == depositstatus.Creating ||
				(claimed.ClaimedFromStatus == depositstatus.ReconciliationRequired &&
					tx.LastProviderErrorCode != nil &&
					(*tx.LastProviderErrorCode == "PROVIDER_LOOKUP_ERROR" || *tx.LastProviderErrorCode == "CREATE_TIMEOUT" || *tx.LastProviderErrorCode == "CREATE_UNKNOWN"))

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

				rowsAffected, err := w.repo.UpdateWithToken(ctx, orderCode, token, updates)
				if err != nil {
					w.log.Error("Failed to recover transaction to Pending", zap.Int64("orderCode", orderCode), zap.Error(err))
				} else if rowsAffected == 0 {
					w.handleZeroRowsAffected(ctx, orderCode, token)
				}
				return
			}

			// If it's cancel timeout error, try cancel again
			if claimed.ClaimedFromStatus == depositstatus.ReconciliationRequired &&
				tx.LastProviderErrorCode != nil &&
				*tx.LastProviderErrorCode == "CANCEL_UNKNOWN" {
				if err := w.gateway.CancelPaymentLink(ctx, orderCode, "Payment link expired"); err != nil {
					w.deferRetry(ctx, orderCode, token, "CANCEL_UNKNOWN")
					return
				}
				w.finish(ctx, orderCode, token, depositstatus.Expired)
				return
			}
		}

		w.deferRetry(ctx, orderCode, token, "PENDING_RETRY")

	case payment.ProviderCancelled:
		w.finish(ctx, orderCode, token, depositstatus.Cancelled)
	default:
		w.deferRetry(ctx, orderCode, token, "UNKNOWN_PROVIDER_STATUS")
	}
}

func (w *PaymentReconciliationWorker) handleZeroRowsAffected(ctx context.Context, orderCode int64, token uuid.UUID) {
	row, err := w.repo.GetByOrderCode(ctx, orderCode)
	if err != nil || row == nil {
		return
	}
	if row.Status == depositstatus.Success {
		w.log.Info("Transaction was already marked Success by Webhook. Worker gracefully exits.", zap.Int64("orderCode", orderCode))
		return
	}
	if row.ProcessingToken == nil || *row.ProcessingToken != token {
		w.log.Warn("Worker became a zombie for transaction. Another worker claimed it.", zap.Int64("orderCode", orderCode))
		return
	}
}

func (w *PaymentReconciliationWorker) finish(ctx context.Context, orderCode int64, token uuid.UUID, status depositstatus.DepositStatus) {
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

	rowsAffected, err := w.repo.UpdateWithToken(ctx, orderCode, token, updates)
	if err != nil {
		w.log.Error("Failed to update finished transaction", zap.Int64("orderCode", orderCode), zap.Error(err))
	} else if rowsAffected == 0 {
		w.handleZeroRowsAffected(ctx, orderCode, token)
	}
}

func (w *PaymentReconciliationWorker) deferRetry(ctx context.Context, orderCode int64, token uuid.UUID, code string) {
	row, err := w.repo.GetByOrderCode(ctx, orderCode)
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

	if row.ReconciliationAttempts >= w.cfg.PayOS.ReconciliationMaxAttempts || now.Sub(row.CreatedAt) >= time.Duration(w.cfg.PayOS.ReconciliationMaxAgeHours)*time.Hour {
		reason := "MAX_RECONCILIATION_POLICY_EXCEEDED"
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

	rowsAffected, err := w.repo.UpdateWithToken(ctx, orderCode, token, updates)
	if err != nil {
		w.log.Error("Failed to defer retry for transaction", zap.Int64("orderCode", orderCode), zap.Error(err))
	} else if rowsAffected == 0 {
		w.handleZeroRowsAffected(ctx, orderCode, token)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
