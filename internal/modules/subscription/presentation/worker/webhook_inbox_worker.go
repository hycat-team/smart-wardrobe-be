package worker

import (
	"context"
	"time"

	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type IWebhookInboxWorker interface {
	Start()
	Stop()
}

type WebhookInboxWorker struct {
	useCase usecase_interfaces.IWebhookInboxUseCase
	cron    *cron.Cron
	log     logger.Interface
}

func NewWebhookInboxWorker(
	useCase usecase_interfaces.IWebhookInboxUseCase,
	log logger.Interface,
) IWebhookInboxWorker {
	return &WebhookInboxWorker{
		useCase: useCase,
		cron:    cron.New(cron.WithSeconds()),
		log:     log,
	}
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

	run := workerlog.New("webhook_inbox", workerlog.TriggerCron)
	if err := w.useCase.ProcessInbox(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log)
}
