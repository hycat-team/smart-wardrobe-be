package worker

import (
	"context"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
	"time"
)

type IWebhookInboxWorker interface {
	Start()
	Stop()
}
type WebhookInboxWorker struct {
	repo       repositories.IWebhookInboxRepository
	gateway    payment.IPaymentGatewayService
	completion usecase_interfaces.IPaymentWebhookUseCase
	cron       *cron.Cron
	log        logger.Interface
}

func NewWebhookInboxWorker(repo repositories.IWebhookInboxRepository, gateway payment.IPaymentGatewayService, completion usecase_interfaces.IPaymentWebhookUseCase, log logger.Interface) IWebhookInboxWorker {
	return &WebhookInboxWorker{repo: repo, gateway: gateway, completion: completion, cron: cron.New(cron.WithSeconds()), log: log}
}
func (w *WebhookInboxWorker) Start() {
	_, err := w.cron.AddFunc("30 */1 * * * *", w.run)
	if err != nil {
		w.log.Error("Failed to register webhook inbox worker", zap.Error(err))
		return
	}
	w.cron.Start()
}
func (w *WebhookInboxWorker) Stop() {
	if w.cron != nil {
		w.cron.Stop()
	}
}
func (w *WebhookInboxWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rows, err := w.repo.Claim(ctx, 50, 30*time.Second)
	if err != nil {
		w.log.Error("Failed to claim webhook inbox events", zap.Error(err))
		return
	}
	for _, row := range rows {
		info, err := w.gateway.GetPaymentLinkInfo(ctx, row.OrderCode)
		if err != nil {
			_ = w.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "provider lookup failed")
			continue
		}
		if info.Status != payment.ProviderPaid {
			_ = w.repo.MarkInvestigation(ctx, row.ID, time.Now().UTC(), "provider does not confirm paid status")
			continue
		}
		if err := w.completion.CompleteVerifiedPayment(ctx, info); err != nil {
			_ = w.repo.MarkRetry(ctx, row.ID, time.Now().UTC(), "payment completion failed")
			continue
		}
		_ = w.repo.MarkProcessed(ctx, row.ID, time.Now().UTC())
	}
}
