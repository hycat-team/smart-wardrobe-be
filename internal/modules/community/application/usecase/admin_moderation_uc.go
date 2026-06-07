package usecase

import (
	"context"

	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AdminCommunityModerationUseCase struct {
	postRepo     repositories.IPostRepository
	postItemRepo repositories.IPostItemRepository
	commentRepo  repositories.ICommentRepository
	wardrobeCtr  wardrobe_contract.IWardrobeContract
	uow          shared_repos.IUnitOfWork
	logger       logger.Interface
}

func NewAdminCommunityModerationUseCase(
	log logger.Interface,
	postRepo repositories.IPostRepository,
	postItemRepo repositories.IPostItemRepository,
	commentRepo repositories.ICommentRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IAdminCommunityModerationUseCase {
	return &AdminCommunityModerationUseCase{
		postRepo:     postRepo,
		postItemRepo: postItemRepo,
		commentRepo:  commentRepo,
		wardrobeCtr:  wardrobeCtr,
		uow:          uow,
		logger:       log,
	}
}

func (uc *AdminCommunityModerationUseCase) AdminDeletePost(ctx context.Context, adminUserID uuid.UUID, postPublicID string) error {
	post, err := uc.postRepo.GetByPublicID(ctx, postPublicID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.postItemRepo.GetByPostID(ctx, post.ID)
	if err != nil {
		return err
	}

	deletePost := func(txCtx context.Context) error {
		if err := uc.postRepo.SoftDelete(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range uniqueItemIDs(postItems) {
			if err := syncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		uc.logger.Info("[CommunityModeration] Admin deleted post",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_post"),
			zap.String("target_type", "post"),
			zap.String("target_id", postPublicID),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deletePost)
}

func (uc *AdminCommunityModerationUseCase) AdminHidePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil {
		return communityerrors.ErrPostItemNotFound
	}

	hidePostItem := func(txCtx context.Context) error {
		postItem.Status = postitemstatus.Hidden
		if err := uc.postItemRepo.Update(txCtx, postItem); err != nil {
			return err
		}
		if err := syncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, postItem.ItemID); err != nil {
			return err
		}
		if err := syncPostTotalPrice(txCtx, uc.postRepo, uc.postItemRepo, postItem.PostID); err != nil {
			return err
		}

		uc.logger.Info("[CommunityModeration] Admin hid post item",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "hide_post_item"),
			zap.String("target_type", "post_item"),
			zap.String("target_id", postItemID.String()),
		)
		return nil
	}

	return uc.uow.Execute(ctx, hidePostItem)
}

func (uc *AdminCommunityModerationUseCase) AdminDeletePostItem(ctx context.Context, adminUserID uuid.UUID, postItemID uuid.UUID) error {
	postItem, err := uc.postItemRepo.GetByID(ctx, postItemID)
	if err != nil {
		return err
	}
	if postItem == nil {
		return communityerrors.ErrPostItemNotFound
	}

	post, err := uc.postRepo.GetByID(ctx, postItem.PostID)
	if err != nil {
		return err
	}
	if post == nil {
		return communityerrors.ErrPostNotFound
	}

	deletePost := func(txCtx context.Context) error {
		postItems, err := uc.postItemRepo.GetByPostID(txCtx, post.ID)
		if err != nil {
			return err
		}

		if err := uc.postRepo.SoftDelete(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range uniqueItemIDs(postItems) {
			if err := syncWardrobeStatusByItem(txCtx, uc.postItemRepo, uc.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		uc.logger.Info("[CommunityModeration] Admin deleted post by post item moderation",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_post_item"),
			zap.String("target_type", "post_item"),
			zap.String("target_id", postItemID.String()),
			zap.String("post_id", post.PublicID),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deletePost)
}

func (uc *AdminCommunityModerationUseCase) AdminDeleteComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error {
	deleteComment := func(txCtx context.Context) error {
		comment, err := uc.commentRepo.GetByID(txCtx, commentID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound
		}

		post, err := uc.postRepo.GetByID(txCtx, comment.PostID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		if err := uc.commentRepo.SoftDelete(txCtx, commentID); err != nil {
			return err
		}

		decrement := 1
		if comment.ParentCommentID == nil {
			replies, err := uc.commentRepo.GetRepliesByParentID(txCtx, post.ID, commentID)
			if err != nil {
				return err
			}
			decrement += len(replies)
			if err := uc.commentRepo.SoftDeleteByParentID(txCtx, commentID); err != nil {
				return err
			}
		}

		post.CommentCount -= decrement
		if post.CommentCount < 0 {
			post.CommentCount = 0
		}
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		if err := uc.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
			return err
		}

		uc.logger.Info("[CommunityModeration] Admin deleted comment",
			zap.String("admin_user_id", adminUserID.String()),
			zap.String("action", "delete_comment"),
			zap.String("target_type", "comment"),
			zap.String("target_id", commentID.String()),
		)
		return nil
	}

	return uc.uow.Execute(ctx, deleteComment)
}

var _ uc_interfaces.IAdminCommunityModerationUseCase = (*AdminCommunityModerationUseCase)(nil)
