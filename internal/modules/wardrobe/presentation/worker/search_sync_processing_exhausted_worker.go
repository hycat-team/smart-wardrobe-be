package worker

import (
	"context"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_msg "smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/internal/shared/presentation/worker"

	"go.uber.org/zap"
)

type ElasticsearchProcessingExhaustedWorker struct {
	*worker.BaseWorker
	messageClient *shared_msg.RabbitMQClient
	useCase       uc_interfaces.ISearchSyncUseCase
	logger        logger.Interface
}

func NewElasticsearchProcessingExhaustedWorker(
	messageClient *shared_msg.RabbitMQClient,
	useCase uc_interfaces.ISearchSyncUseCase,
	l logger.Interface,
) *ElasticsearchProcessingExhaustedWorker {
	return &ElasticsearchProcessingExhaustedWorker{
		BaseWorker:    worker.NewBaseWorker("elasticsearch_processing_exhausted"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *ElasticsearchProcessingExhaustedWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := shared_msg.RegisterTypedConsumer(w.messageClient, shared_msg.QueueElasticsearchProcessingExhausted, w.handle, shared_msg.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register elasticsearch processing exhausted consumer", zap.Error(err))
	}
}

func (w *ElasticsearchProcessingExhaustedWorker) handle(ctx context.Context, payload fashionDto.FashionProcessingExhaustedPayload) error {
	run := workerlog.New("elasticsearch_processing_exhausted", workerlog.TriggerQueue)
	run.LogReceived(w.logger,
		zap.String("correlationId", shared_msg.CorrelationIdFromContext(ctx)),
		zap.String("wardrobeItemId", payload.WardrobeItemID.String()),
	)
	if err := w.useCase.HandleFashionProcessingExhausted(ctx, payload, run); err != nil {
		run.LogFailure(w.logger, err,
			zap.String("wardrobeItemId", payload.WardrobeItemID.String()),
			zap.Bool("willRequeue", w.useCase.IsOfflineError(err)),
		)
		return err
	}
	run.LogSuccessIfData(w.logger, zap.String("wardrobeItemId", payload.WardrobeItemID.String()))
	return nil
}

func (w *ElasticsearchProcessingExhaustedWorker) Stop() {
	w.BaseWorker.Stop()
}
