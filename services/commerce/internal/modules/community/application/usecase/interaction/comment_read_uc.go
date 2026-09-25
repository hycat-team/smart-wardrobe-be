package interaction

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *InteractionUseCase) GetPostComments(ctx context.Context, publicID string) ([]*dto.CommentRes, error) {
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return nil, err
	}
	roots, err := uc.commentRepo.GetRoots(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	rootIDs := make([]uuid.UUID, 0, len(roots))
	userIDs := make([]uuid.UUID, 0, len(roots))
	for _, root := range roots {
		rootIDs = append(rootIDs, root.ID)
		userIDs = append(userIDs, root.UserID)
	}
	counts, err := uc.commentRepo.CountRepliesByRootIDs(ctx, rootIDs)
	if err != nil {
		return nil, err
	}
	authors := uc.fetchAuthors(ctx, userIDs)

	result := make([]*dto.CommentRes, 0, len(roots))
	for _, root := range roots {
		replyCount := counts[root.ID]
		if root.Status == poststatus.Deleted && replyCount == 0 {
			continue
		}
		result = append(result, mapper.MapToCommentRes(root, authors[root.UserID], replyCount))
	}
	return result, nil
}

func (uc *InteractionUseCase) GetCommentReplies(ctx context.Context, publicID string, rootCommentID uuid.UUID) ([]*dto.CommentRes, error) {
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return nil, err
	}
	root, err := uc.commentRepo.GetByID(ctx, rootCommentID)
	if err != nil {
		return nil, err
	}
	if root == nil || root.PostID != post.ID {
		return []*dto.CommentRes{}, nil
	}

	replies, err := uc.commentRepo.GetReplies(ctx, rootCommentID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uuid.UUID, 0, len(replies))
	for _, reply := range replies {
		userIDs = append(userIDs, reply.UserID)
	}
	authors := uc.fetchAuthors(ctx, userIDs)

	result := make([]*dto.CommentRes, 0, len(replies))
	for _, reply := range replies {
		result = append(result, mapper.MapToCommentRes(reply, authors[reply.UserID], 0))
	}
	return result, nil
}

func (uc *InteractionUseCase) GetPostLikes(ctx context.Context, publicID string, query dto.GetLikesQueryReq) (*dto.PostLikeUserListRes, error) {
	post, err := uc.getPublishedPost(ctx, publicID)
	if err != nil {
		return nil, err
	}
	likes, total, err := uc.likeRepo.GetByPostIDPaginated(ctx, post.ID, query.PaginationQuery)
	if err != nil {
		return nil, err
	}

	userIDs := make([]uuid.UUID, 0, len(likes))
	for _, like := range likes {
		userIDs = append(userIDs, like.UserID)
	}
	authors := uc.fetchAuthors(ctx, userIDs)

	items := make([]*dto.CommunityUserRes, 0, len(likes))
	for _, like := range likes {
		if author := authors[like.UserID]; author != nil {
			items = append(items, mapper.MapToPostLikeUserRes(author))
		}
	}

	return &dto.PostLikeUserListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, total),
	}, nil
}
