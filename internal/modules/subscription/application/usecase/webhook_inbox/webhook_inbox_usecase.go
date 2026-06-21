package webhook_inbox

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
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

func (uc *WebhookInboxUseCase) ProcessInbox(ctx context.Context) error {
	rows, err := uc.repo.Claim(ctx, 50, 30*time.Second)
	if err != nil {
		uc.log.Error("Failed to claim webhook inbox events", zap.Error(err))
		return err
	}
	for _, row := range rows {
		info, err := uc.gateway.GetPaymentLinkInfo(ctx, row.OrderCode)
		if err != nil {
			_ = uc.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "provider lookup failed")
			continue
		}
		if info.Status != payment.ProviderPaid {
			_ = uc.repo.MarkInvestigation(ctx, row.ID, time.Now().UTC(), "provider does not confirm paid status")
			continue
		}
		if err := uc.completion.CompleteVerifiedPayment(ctx, info); err != nil {
			_ = uc.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "payment completion failed")
			continue
		}
		_ = uc.repo.MarkProcessed(ctx, row.ID, time.Now().UTC())
	}
	return nil
}
