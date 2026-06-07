package community

import (
	"smart-wardrobe-be/internal/modules/community/application/usecase"
	"smart-wardrobe-be/internal/modules/community/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/community/presentation/handler"
	"smart-wardrobe-be/internal/modules/community/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewPostRepository,
	persistence.NewPostScoreRepository,
	persistence.NewPostItemRepository,
	persistence.NewPostMediaRepository,
	persistence.NewCommentRepository,
	persistence.NewLikeRepository,
	persistence.NewTransferRequestRepository,
	usecase.NewUserPostUseCase,
	usecase.NewPostInteractionUseCase,
	usecase.NewAdminCommunityModerationUseCase,
	usecase.NewItemTransferUseCase,
	handler.NewAdminHandler,
	handler.NewPostHandler,
	handler.NewPostInteractionHandler,
	handler.NewItemTransferHandler,
	worker.NewPostHotnessWorker,
)
