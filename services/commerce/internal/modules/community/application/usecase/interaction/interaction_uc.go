package interaction

import (
	"context"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_caching "smart-wardrobe-be/shared/application/interface/caching"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

type InteractionUseCase struct {
	cfg         *config.Config
	logger      logger.Interface
	uow         shared_repos.IUnitOfWork
	postRepo    repositories.IPostRepository
	commentRepo repositories.IPostCommentRepository
	likeRepo    repositories.IPostLikeRepository
	userBrief   identitycontract.IUserBriefContract
	rateLimiter shared_caching.IRateLimiter
}

func NewInteractionUseCase(
	cfg *config.Config,
	log logger.Interface,
	uow shared_repos.IUnitOfWork,
	postRepo repositories.IPostRepository,
	commentRepo repositories.IPostCommentRepository,
	likeRepo repositories.IPostLikeRepository,
	userBrief identitycontract.IUserBriefContract,
	rateLimiter shared_caching.IRateLimiter,
) uc_interfaces.IPostInteractionUseCase {
	return &InteractionUseCase{
		cfg:         cfg,
		logger:      log,
		uow:         uow,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
		userBrief:   userBrief,
		rateLimiter: rateLimiter,
	}
}

func (uc *InteractionUseCase) TogglePostLike(ctx context.Context, userID uuid.UUID, publicID string, isLiked bool) error {
	if err := uc.checkRateLimit(ctx, "like", userID, uc.cfg.Community.LikeRateLimitPerHour); err != nil {
		return err
	}
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return err
	}

	exists, err := uc.likeRepo.Exists(ctx, post.ID, userID)
	if err != nil {
		return err
	}
	if isLiked == exists {
		return nil
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if isLiked {
			if err := uc.likeRepo.Create(txCtx, &entities.PostLike{PostID: post.ID, UserID: userID}); err != nil {
				return err
			}
			return uc.postRepo.IncrementLikeCount(txCtx, post.ID, 1)
		}
		affected, err := uc.likeRepo.Delete(txCtx, post.ID, userID)
		if err != nil {
			return err
		}
		if affected == 0 {
			// A concurrent unlike already removed the row and decremented the
			// counter; this transaction must not decrement it a second time.
			return nil
		}
		return uc.postRepo.IncrementLikeCount(txCtx, post.ID, -1)
	})
}

func (uc *InteractionUseCase) getPublishedPost(ctx context.Context, publicID string) (*entities.Post, error) {
	post, err := uc.postRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status != poststatus.Published {
		return nil, communityerrors.ErrPostNotFound()
	}
	return post, nil
}

func (uc *InteractionUseCase) checkRateLimit(ctx context.Context, action string, userID uuid.UUID, limit int) error {
	if uc.rateLimiter == nil || limit <= 0 {
		return nil
	}
	allowed, err := uc.rateLimiter.Allow(ctx, "community:"+action+":"+userID.String(), limit, 3600)
	if err != nil || !allowed {
		return communityerrors.ErrRateLimited()
	}
	return nil
}
