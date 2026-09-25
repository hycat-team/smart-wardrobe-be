package community

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/admin"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/event"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/follow"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/interaction"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/post"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/profile"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/search"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/statsync"
	"smart-wardrobe-be/services/commerce/internal/modules/community/infrastructure/persistence"
	"smart-wardrobe-be/services/commerce/internal/modules/community/presentation/handler"
	"smart-wardrobe-be/services/commerce/internal/modules/community/presentation/router"
	presentation_worker "smart-wardrobe-be/services/commerce/internal/modules/community/presentation/worker"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	persistence.NewPostRepository,
	persistence.NewPostMediaRepository,
	persistence.NewPostLikeRepository,
	persistence.NewPostCommentRepository,
	persistence.NewFollowRepository,
	persistence.NewUserFollowStatsRepository,

	post.NewPostUseCase,
	interaction.NewInteractionUseCase,
	follow.NewFollowUseCase,
	profile.NewProfileUseCase,
	search.NewSearchUseCase,
	admin.NewAdminUseCase,
	statsync.NewStatsSyncUseCase,
	event.NewCommunityEventUseCase,

	handler.NewPostHandler,
	handler.NewInteractionHandler,
	handler.NewFollowHandler,
	handler.NewProfileHandler,
	handler.NewSearchHandler,
	handler.NewAdminHandler,
	router.NewRouter,
	presentation_worker.NewOutfitCascadeWorker,
	presentation_worker.NewUserStatusWorker,
	presentation_worker.NewFollowStatsReconcileWorker,
)
