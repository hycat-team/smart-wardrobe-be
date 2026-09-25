package worker

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/presentation/worker"

	"go.uber.org/zap"
)

type OutfitCascadeWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.ICommunityEventUseCase
	logger        logger.Interface
}

func NewOutfitCascadeWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.ICommunityEventUseCase,
	l logger.Interface,
) *OutfitCascadeWorker {
	return &OutfitCascadeWorker{
		BaseWorker:    worker.NewBaseWorker("community_outfit_cascade"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *OutfitCascadeWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueCommunityOutfitCascade, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register community outfit cascade consumer", zap.Error(err))
	}
}

func (w *OutfitCascadeWorker) handle(ctx context.Context, payload dto.OutfitDeletedEventPayload) error {
	if err := w.useCase.HandleOutfitDeleted(ctx, payload); err != nil {
		w.logger.Error("Failed to cascade outfit deletion",
			zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
			zap.String("outfitId", payload.OutfitID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (w *OutfitCascadeWorker) Stop() {
	w.BaseWorker.Stop()
}
