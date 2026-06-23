package webhook_inbox

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type WebhookInboxUseCase struct {
	repo       repositories.IWebhookInboxRepository
	gateway    payment.IPaymentGatewayService
	completion usecase_interfaces.IPaymentWebhookUseCase
	log        logger.Interface
}

func NewWebhookInboxUseCase(
	repo repositories.IWebhookInboxRepository,
	gateway payment.IPaymentGatewayService,
	completion usecase_interfaces.IPaymentWebhookUseCase,
	log logger.Interface,
) usecase_interfaces.IWebhookInboxUseCase {
	return &WebhookInboxUseCase{
		repo:       repo,
		gateway:    gateway,
		completion: completion,
		log:        log,
	}
}

func (uc *WebhookInboxUseCase) ProcessInbox(ctx context.Context, run *workerlog.Run) error {
	rows, err := uc.repo.Claim(ctx, 50, 30*time.Second)
	if err != nil {
		run.ChildError(uc.log, "Failed to claim webhook inbox events", zap.Error(err))
		return err
	}
	retryMarkedCount := 0
	investigationMarkedCount := 0
	processedCount := 0
	for _, row := range rows {
		run.AddTotal(1)
		info, err := uc.gateway.GetPaymentLinkInfo(ctx, row.OrderCode)
		if err != nil {
			_ = uc.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "provider lookup failed")
			run.ChildWarn(uc.log, "Provider lookup failed for webhook inbox event",
				zap.String("inboxId", row.ID.String()),
				zap.Int64("orderCode", row.OrderCode),
				zap.String("actionTaken", "mark_retry"),
				zap.Error(err),
			)
			retryMarkedCount++
			run.AddRetry(1)
			continue
		}
		if info.Status != payment.ProviderPaid {
			_ = uc.repo.MarkInvestigation(ctx, row.ID, time.Now().UTC(), "provider does not confirm paid status")
			run.ChildWarn(uc.log, "Webhook inbox event marked for investigation",
				zap.String("inboxId", row.ID.String()),
				zap.Int64("orderCode", row.OrderCode),
				zap.String("actionTaken", "mark_investigation"),
				zap.String("providerStatus", string(info.Status)),
			)
			investigationMarkedCount++
			run.AddSkipped(1)
			continue
		}
		if err := uc.completion.CompleteVerifiedPayment(ctx, info); err != nil {
			_ = uc.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "payment completion failed")
			run.ChildWarn(uc.log, "Payment completion failed for webhook inbox event",
				zap.String("inboxId", row.ID.String()),
				zap.Int64("orderCode", row.OrderCode),
				zap.String("actionTaken", "mark_retry"),
				zap.Error(err),
			)
			retryMarkedCount++
			run.AddRetry(1)
			continue
		}
		_ = uc.repo.MarkProcessed(ctx, row.ID, time.Now().UTC())
		processedCount++
		run.AddSuccess(1)
	}
	run.AddSummaryFields(
		zap.Int("retryMarkedCount", retryMarkedCount),
		zap.Int("investigationMarkedCount", investigationMarkedCount),
		zap.Int("processedCount", processedCount),
	)
	return nil
}
