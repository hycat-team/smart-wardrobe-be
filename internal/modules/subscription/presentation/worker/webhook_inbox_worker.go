package worker

import (
	"context"
	"time"

	usecase_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	shared_worker "smart-wardrobe-be/internal/shared/presentation/worker"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type WebhookInboxWorker struct {
	*shared_worker.BaseWorker
	useCase usecase_interfaces.IWebhookInboxUseCase
	cron    *cron.Cron
	log     logger.Interface
}

func NewWebhookInboxWorker(
	useCase usecase_interfaces.IWebhookInboxUseCase,
	log logger.Interface,
) *WebhookInboxWorker {
	return &WebhookInboxWorker{
		BaseWorker: shared_worker.NewBaseWorker("webhook_inbox"),
		useCase:    useCase,
		cron:       cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *WebhookInboxWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)

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
	w.BaseWorker.Stop()
}

func (w *WebhookInboxWorker) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	run := workerlog.New("webhook_inbox", workerlog.TriggerCron)
	if err := w.useCase.ProcessInbox(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccessIfData(w.log)
}
