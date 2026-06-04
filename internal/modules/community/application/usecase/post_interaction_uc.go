package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
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
) uc_interfaces.IPostInteractionUseCase {
	return &PostInteractionUseCase{
		postRepo:    postRepo,
		commentRepo: commentRepo,
		likeRepo:    likeRepo,
		uow:         uow,
	}
}

func (uc *PostInteractionUseCase) TogglePostLike(ctx context.Context, userID uuid.UUID, postID uuid.UUID, isLiked bool) error {
	togglePostLike := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(txCtx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return errorcode.NewNotFound("Không tìm thấy bài đăng.")
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

			return uc.postRepo.Update(txCtx, post)
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

		return uc.postRepo.Update(txCtx, post)
	}

	return uc.uow.Execute(ctx, togglePostLike)
}

func (uc *PostInteractionUseCase) AddComment(ctx context.Context, userID uuid.UUID, postID uuid.UUID, content string) (*dto.CommentRes, error) {
	var comment *entities.Comment

	addComment := func(txCtx context.Context) error {
		post, err := uc.postRepo.GetByID(ctx, postID)
		if err != nil {
			return err
		}
		if post == nil {
			return errorcode.NewNotFound("Không tìm thấy bài đăng.")
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
		return nil
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

var _ uc_interfaces.IPostInteractionUseCase = (*PostInteractionUseCase)(nil)
