package worker

import (
	"context"
	"time"

	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type IPostHotnessWorker interface {
	Start()
	Stop()
}

type PostHotnessWorker struct {
	useCase    usecase_interfaces.IPostHotnessUseCase
	cronEngine *cron.Cron
	log        logger.Interface
}

func NewPostHotnessWorker(
	useCase usecase_interfaces.IPostHotnessUseCase,
	log logger.Interface,
) IPostHotnessWorker {
	return &PostHotnessWorker{
		useCase:    useCase,
		cronEngine: cron.New(cron.WithSeconds()),
		log:        log,
	}
}

func (w *PostHotnessWorker) Start() {
	go w.executeRefresh(workerlog.TriggerStartup)

	_, err := w.cronEngine.AddFunc("0 */10 * * * *", func() {
		w.executeRefresh(workerlog.TriggerCron)
	})
	if err != nil {
		w.log.Error("Failed to register community hotness worker", zap.Error(err))
		return
	}

	w.cronEngine.Start()
}

func (w *PostHotnessWorker) Stop() {
	if w.cronEngine != nil {
		w.cronEngine.Stop()
	}
}

func (w *PostHotnessWorker) executeRefresh(triggerType string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	run := workerlog.New("post_hotness", triggerType)
	if err := w.useCase.RefreshHotness(ctx, run); err != nil {
		run.LogFailure(w.log, err)
		return
	}
	run.LogSuccess(w.log)
}
