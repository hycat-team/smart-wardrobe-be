package post_interaction

import (
	"context"

	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/application/usecase/post"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type PostInteractionUseCase struct {
	postRepo    repositories.IPostRepository
	commentRepo repositories.ICommentRepository
	likeRepo    repositories.ILikeRepository
	uow         shared_repos.IUnitOfWork
}

func NewPostInteractionUseCase(
	postRepo repositories.IPostRepository,
	commentRepo repositories.ICommentRepository,
	likeRepo repositories.ILikeRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IUserPostInteractionUseCase {
	return &PostInteractionUseCase{
		postRepo:    postRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
		uow:         uow,
	}
}

func (uc *PostInteractionUseCase) TogglePostLike(ctx context.Context, userID uuid.UUID, postPublicID string, isLiked bool) error {
	normalizedPublicID, err := post.NormalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	togglePostLike := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByPublicID(txCtx, normalizedPublicID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound()
		}

		like, err := uc.likeRepo.GetPostLike(txCtx, userID, post.ID)
		if err != nil {
			return err
		}

		if isLiked {
			if like != nil {
				return nil
			}

			post.LikeCount++
			postIDCopy := post.ID
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

func (uc *PostInteractionUseCase) AddComment(ctx context.Context, userID uuid.UUID, postPublicID string, input community_dto.AddCommentReq) (*community_dto.CommentRes, error) {
	normalizedPublicID, err := post.NormalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	var comment *entities.Comment

	addComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByPublicID(txCtx, normalizedPublicID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound()
		}

		if input.ParentCommentID != nil {
			parentComment, err := uc.commentRepo.GetByIDAndPostID(txCtx, *input.ParentCommentID, post.ID)
			if err != nil {
				return err
			}
			if parentComment == nil || parentComment.ParentCommentID != nil {
				return communityerrors.ErrCommentReplyTargetInvalid()
			}
		}

		comment = &entities.Comment{
			PostID:          post.ID,
			UserID:          userID,
			ParentCommentID: input.ParentCommentID,
			Content:         input.Content,
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

	comment, err = uc.commentRepo.GetByID(ctx, comment.ID)
	if err != nil {
		return nil, err
	}
	return mapper.MapCommentRes(comment), nil
}

func (uc *PostInteractionUseCase) UpdateComment(ctx context.Context, userID uuid.UUID, postPublicID string, commentID uuid.UUID, content string) (*community_dto.CommentRes, error) {
	normalizedPublicID, err := post.NormalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	var comment *entities.Comment

	updateComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByPublicID(txCtx, normalizedPublicID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound()
		}

		comment, err = uc.commentRepo.GetByIDAndPostID(txCtx, commentID, post.ID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound()
		}
		if comment.UserID != userID {
			return communityerrors.ErrEditCommentForbidden()
		}

		comment.Content = content
		return uc.commentRepo.Update(txCtx, comment)
	}

	if err := uc.uow.Execute(ctx, updateComment); err != nil {
		return nil, err
	}

	comment, err = uc.commentRepo.GetByID(ctx, comment.ID)
	if err != nil {
		return nil, err
	}
	return mapper.MapCommentRes(comment), nil
}

func (uc *PostInteractionUseCase) DeleteComment(ctx context.Context, userID uuid.UUID, postPublicID string, commentID uuid.UUID) error {
	normalizedPublicID, err := post.NormalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	deleteComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByPublicID(txCtx, normalizedPublicID)
		if err != nil {
			return err
		}
		if post == nil {
			return communityerrors.ErrPostNotFound()
		}

		comment, err := uc.commentRepo.GetByIDAndPostID(txCtx, commentID, post.ID)
		if err != nil {
			return err
		}
		if comment == nil {
			return communityerrors.ErrCommentNotFound()
		}
		if comment.UserID != userID {
			return communityerrors.ErrDeleteCommentForbidden()
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
			if len(replies) > 0 {
				decrement += len(replies)
				if err := uc.commentRepo.SoftDeleteByParentID(txCtx, commentID); err != nil {
					return err
				}
			}
		}

		post.CommentCount -= decrement
		if post.CommentCount < 0 {
			post.CommentCount = 0
		}
		if err := uc.postRepo.Update(txCtx, post); err != nil {
			return err
		}
		return uc.postRepo.MarkHotnessDirty(txCtx, post.ID)
	}

	return uc.uow.Execute(ctx, deleteComment)
}

var _ uc_interfaces.IUserPostInteractionUseCase = (*PostInteractionUseCase)(nil)
