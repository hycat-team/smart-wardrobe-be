package post

import (
	"smart-wardrobe-be/config"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/contract"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
)

const (
	defaultFeedLimit = 20
)

type postFeedDependencies struct {
	postRepo      repositories.IPostRepository
	postScoreRepo repositories.IPostScoreRepository
	postItemRepo  repositories.IPostItemRepository
	postMediaRepo repositories.IPostMediaRepository
	commentRepo   repositories.ICommentRepository
	likeRepo      repositories.ILikeRepository
	identityCtr   identity_contract.IUserContract
}

type postPublishingDependencies struct {
	postRepo      repositories.IPostRepository
	postItemRepo  repositories.IPostItemRepository
	postMediaRepo repositories.IPostMediaRepository
	wardrobeCtr   wardrobe_contract.IWardrobeContract
	uow           shared_repos.IUnitOfWork
}

type UserPostUseCase struct {
	cfg          *config.Config
	logger       logger.Interface
	feed         postFeedDependencies
	publishing   postPublishingDependencies
	mediaService media.IMediaService
}

// NewUserPostUseCase builds the community post use case with grouped feed and publishing dependencies.
func NewUserPostUseCase(
	cfg *config.Config,
	log logger.Interface,
	postRepo repositories.IPostRepository,
	postScoreRepo repositories.IPostScoreRepository,
	postItemRepo repositories.IPostItemRepository,
	postMediaRepo repositories.IPostMediaRepository,
	commentRepo repositories.ICommentRepository,
	likeRepo repositories.ILikeRepository,
	identityCtr identity_contract.IUserContract,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	mediaService media.IMediaService,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IUserPostUseCase {
	return &UserPostUseCase{
		cfg:    cfg,
		logger: log,
		feed: postFeedDependencies{
			postRepo:      postRepo,
			postScoreRepo: postScoreRepo,
			postItemRepo:  postItemRepo,
			postMediaRepo: postMediaRepo,
			commentRepo:   commentRepo,
			likeRepo:      likeRepo,
			identityCtr:   identityCtr,
		},
		publishing: postPublishingDependencies{
			postRepo:      postRepo,
			postItemRepo:  postItemRepo,
			postMediaRepo: postMediaRepo,
			wardrobeCtr:   wardrobeCtr,
			uow:           uow,
		},
		mediaService: mediaService,
	}
}

var _ uc_interfaces.IUserPostUseCase = (*UserPostUseCase)(nil)
