package community

import (
	"smart-wardrobe-be/internal/modules/community/application/usecase"
	"smart-wardrobe-be/internal/modules/community/infrastructure/persistence"
	"smart-wardrobe-be/internal/modules/community/presentation/handler"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewPostRepository,
	persistence.NewPostItemRepository,
	persistence.NewPostMediaRepository,
	persistence.NewCommentRepository,
	persistence.NewLikeRepository,
	usecase.NewPostUseCase,
	usecase.NewPostInteractionUseCase,
	usecase.NewItemTransferUseCase,
	handler.NewPostHandler,
	handler.NewPostInteractionHandler,
	handler.NewItemTransferHandler,
)
