package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/modules/community/application/errors"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PostInteractionUseCase struct {
	postRepo    repositories.IPostRepository
	commentRepo repositories.ICommentRepository
	likeRepo    repositories.ILikeRepository
	uow         shared_repos.IUnitOfWork
	logger      logger.Interface
}

func NewPostInteractionUseCase(
	log logger.Interface,
	postRepo repositories.IPostRepository,
	commentRepo repositories.ICommentRepository,
	likeRepo repositories.ILikeRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IPostInteractionUseCase {
	return &PostInteractionUseCase{
		postRepo:    postRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
		uow:         uow,
		logger:      log,
	}
}

func (uc *PostInteractionUseCase) TogglePostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID, isLiked bool) error {
	togglePostLike := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(txCtx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		like, err := uc.likeRepo.GetPostLike(txCtx, userID, postID)
		if err != nil {
			return err
		}

		if isLiked {
			if like != nil {
				return nil
			}

			post.LikeCount++
			postIDCopy := postID
			if err := uc.likeRepo.Create(txCtx, &entities.Like{
				UserID: userID,
				PostID: &postIDCopy,
			}); err != nil {
				return err
			}

			if err := uc.postRepo.Update(txCtx, post); err != nil {
				return err
			}
			return uc.postRepo.MarkHotnessDirty(txCtx, post.ID)
		}

		if like == nil {
			return nil
		}

		if post.LikeCount > 0 {
			post.LikeCount--
		}

		if err := uc.likeRepo.Delete(txCtx, like.ID); err != nil {
			return err
		}

		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		return uc.postRepo.MarkHotnessDirty(txCtx, post.ID)
	}

	return uc.uow.Execute(ctx, togglePostLike)
}

func (uc *PostInteractionUseCase) AddComment(ctx context.Context, userID uuid.UUID, postID uuid.UUID, content string) (*dto.CommentRes, error) {
	var comment *entities.Comment

	addComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(txCtx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		comment = &entities.Comment{
			PostID:  postID,
			UserID:  userID,
			Content: content,
		}

		if err := uc.commentRepo.Create(txCtx, comment); err != nil {
			return err
		}

		post.CommentCount++
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		return uc.postRepo.MarkHotnessDirty(txCtx, post.ID)
	}

	if err := uc.uow.Execute(ctx, addComment); err != nil {
		return nil, err
	}

	return &dto.CommentRes{
		ID:        comment.ID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}, nil
}

func (uc *PostInteractionUseCase) UpdateComment(ctx context.Context, userID uuid.UUID, postID uuid.UUID, commentID uuid.UUID, content string) (*dto.CommentRes, error) {
	var comment *entities.Comment

	updateComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(txCtx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		comment, err = uc.commentRepo.GetByIDAndPostID(txCtx, commentID, postID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound
		}
		if comment.UserID != userID {
			return communityerrors.ErrEditCommentForbidden
		}

		comment.Content = content
		return uc.commentRepo.Update(txCtx, comment)
	}

	if err := uc.uow.Execute(ctx, updateComment); err != nil {
		return nil, err
	}

	return &dto.CommentRes{
		ID:        comment.ID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}, nil
}

func (uc *PostInteractionUseCase) DeleteComment(ctx context.Context, userID uuid.UUID, postID uuid.UUID, commentID uuid.UUID) error {
	deleteComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(txCtx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound
		}

		comment, err := uc.commentRepo.GetByIDAndPostID(txCtx, commentID, postID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound
		}
		if comment.UserID != userID {
			return communityerrors.ErrDeleteCommentForbidden
		}

		if err := uc.commentRepo.Delete(txCtx, commentID); err != nil {
			return err
		}

		if post.CommentCount > 0 {
			post.CommentCount--
		}
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		return uc.postRepo.MarkHotnessDirty(txCtx, post.ID)
	}

	return uc.uow.Execute(ctx, deleteComment)
}

func (uc *PostInteractionUseCase) AdminDeleteComment(ctx context.Context, adminUserID uuid.UUID, commentID uuid.UUID) error {
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

		if err := uc.commentRepo.Delete(txCtx, commentID); err != nil {
			return err
		}

		if post.CommentCount > 0 {
			post.CommentCount--
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

var _ uc_interfaces.IPostInteractionUseCase = (*PostInteractionUseCase)(nil)
