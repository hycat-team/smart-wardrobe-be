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

type UserStatusWorker struct {
	*worker.BaseWorker
	messageClient event.IMessageConsumer
	useCase       uc_interfaces.ICommunityEventUseCase
	logger        logger.Interface
}

func NewUserStatusWorker(
	messageClient event.IMessageConsumer,
	useCase uc_interfaces.ICommunityEventUseCase,
	l logger.Interface,
) *UserStatusWorker {
	return &UserStatusWorker{
		BaseWorker:    worker.NewBaseWorker("community_user_status"),
		messageClient: messageClient,
		useCase:       useCase,
		logger:        l,
	}
}

func (w *UserStatusWorker) Start(ctx context.Context) {
	w.BaseWorker.Start(ctx)
	if err := event.RegisterTypedConsumer(w.messageClient, eventconstants.QueueCommunityUserStatus, w.handle, event.ConsumerOpts{}); err != nil {
		w.logger.Error("Failed to register community user status consumer", zap.Error(err))
	}
}

func (w *UserStatusWorker) handle(ctx context.Context, payload dto.UserStatusChangedEventPayload) error {
	if err := w.useCase.HandleUserStatusChanged(ctx, payload); err != nil {
		w.logger.Error("Failed to apply user status to community posts",
			zap.String("correlationId", event.CorrelationIdFromContext(ctx)),
			zap.String("userId", payload.UserID.String()),
			zap.Bool("active", payload.Active),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (w *UserStatusWorker) Stop() {
	w.BaseWorker.Stop()
}
