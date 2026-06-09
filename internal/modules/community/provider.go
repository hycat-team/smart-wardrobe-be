package community

import (
	"smart-wardrobe-be/internal/modules/community/application/usecase/admin_moderation"
	"smart-wardrobe-be/internal/modules/community/application/usecase/item_transfer"
	"smart-wardrobe-be/internal/modules/community/application/usecase/post"
	"smart-wardrobe-be/internal/modules/community/application/usecase/post_interaction"
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
	post.NewUserPostUseCase,
	post_interaction.NewPostInteractionUseCase,
	admin_moderation.NewAdminCommunityModerationUseCase,
	item_transfer.NewItemTransferUseCase,
	handler.NewAdminHandler,
	handler.NewPostHandler,
	handler.NewPostInteractionHandler,
	handler.NewItemTransferHandler,
	worker.NewPostHotnessWorker,
)
