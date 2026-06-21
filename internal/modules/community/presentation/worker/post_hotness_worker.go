package worker

import (
	"context"
	"time"

	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
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
	go w.executeRefresh()

	_, err := w.cronEngine.AddFunc("0 */10 * * * *", w.executeRefresh)
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

func (w *PostHotnessWorker) executeRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := w.useCase.RefreshHotness(ctx); err != nil {
		w.log.Error("Community hotness calculation failed", zap.Error(err))
	}
}
