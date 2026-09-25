package post

import (
	"context"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	usecase_shared "smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PostUseCase struct {
	cfg            *config.Config
	logger         logger.Interface
	uow            shared_repos.IUnitOfWork
	postRepo       repositories.IPostRepository
	mediaRepo      repositories.IPostMediaRepository
	likeRepo       repositories.IPostLikeRepository
	followRepo     repositories.IFollowRepository
	outfitContract wardrobe_contract.IOutfitContract
	userBrief      identitycontract.IUserBriefContract
	rateLimiter    shared_caching.IRateLimiter
	mediaService   media.IMediaService
}

func NewPostUseCase(
	cfg *config.Config,
	log logger.Interface,
	uow shared_repos.IUnitOfWork,
	postRepo repositories.IPostRepository,
	mediaRepo repositories.IPostMediaRepository,
	likeRepo repositories.IPostLikeRepository,
	followRepo repositories.IFollowRepository,
	outfitContract wardrobe_contract.IOutfitContract,
	userBrief identitycontract.IUserBriefContract,
	rateLimiter shared_caching.IRateLimiter,
	mediaService media.IMediaService,
) uc_interfaces.IPostUseCase {
	return &PostUseCase{
		cfg:            cfg,
		logger:         log,
		uow:            uow,
		postRepo:       postRepo,
		mediaRepo:      mediaRepo,
		likeRepo:       likeRepo,
		followRepo:     followRepo,
		outfitContract: outfitContract,
		userBrief:      userBrief,
		rateLimiter:    rateLimiter,
		mediaService:   mediaService,
	}
}

func (uc *PostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error) {
	if err := uc.checkRateLimit(ctx, "post", userID, uc.cfg.Community.PostRateLimitPerHour); err != nil {
		return nil, err
	}
	if err := uc.validatePostInput(input.PostType, input.Title, input.Content, input.Media); err != nil {
		return nil, err
	}

	var outfit *wardrobe_dto.OutfitContractDTO
	if input.PostType == posttype.Outfit {
		fetched, err := uc.validateOutfitOwnership(ctx, userID, input.OutfitID)
		if err != nil {
			return nil, err
		}
		outfit = fetched
	}

	publicID, err := usecase_shared.GeneratePublicID()
	if err != nil {
		return nil, apperror.NewInternalError("Không thể tạo bài đăng.")
	}

	post := &entities.Post{
		PublicID: publicID,
		UserID:   userID,
		PostType: input.PostType,
		Title:    input.Title,
		Content:  input.Content,
		OutfitID: input.OutfitID,
		Status:   poststatus.Published,
	}
	post.ID = uuid.New()
	medias := buildPostMedia(post.ID, input.Media)

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.postRepo.Create(txCtx, post); err != nil {
			return err
		}
		if len(medias) > 0 {
			if err := uc.mediaRepo.BulkCreate(txCtx, medias); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	post.Media = medias
	return uc.enrichSingle(ctx, post, &userID, outfit)
}

func (uc *PostUseCase) UpdatePost(ctx context.Context, userID uuid.UUID, publicID string, input dto.UpdatePostReq) (*dto.PostRes, error) {
	post, err := uc.postRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status == poststatus.Deleted {
		return nil, communityerrors.ErrPostNotFound()
	}
	if post.UserID != userID {
		return nil, communityerrors.ErrNotPostOwner()
	}
	if err := uc.validatePostInput(post.PostType, input.Title, input.Content, input.Media); err != nil {
		return nil, err
	}

	var outfit *wardrobe_dto.OutfitContractDTO
	if post.PostType == posttype.Outfit {
		fetched, err := uc.validateOutfitOwnership(ctx, userID, input.OutfitID)
		if err != nil {
			return nil, err
		}
		outfit = fetched
		post.OutfitID = input.OutfitID
	}

	post.Title = input.Title
	post.Content = input.Content
	medias := buildPostMedia(post.ID, input.Media)

	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.mediaRepo.DeleteByPostID(txCtx, post.ID); err != nil {
			return err
		}
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		if len(medias) > 0 {
			if err := uc.mediaRepo.BulkCreate(txCtx, medias); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	post.Media = medias
	return uc.enrichSingle(ctx, post, &userID, outfit)
}

func (uc *PostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, publicID string) error {
	post, err := uc.postRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return err
	}
	if post == nil || post.Status == poststatus.Deleted {
		return communityerrors.ErrPostNotFound()
	}
	if post.UserID != userID {
		return communityerrors.ErrNotPostOwner()
	}
	uc.logger.Info("community post deleted by author", zap.String("public_id", publicID))
	return uc.postRepo.UpdateModerationStatus(ctx, post.ID, poststatus.Deleted, nil, nil)
}
