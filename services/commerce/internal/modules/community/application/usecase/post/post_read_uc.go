package post

import (
	"context"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	usecase_shared "smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedsort"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedtype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

func (uc *PostUseCase) GetPostDetail(ctx context.Context, publicID string, viewerUserID *uuid.UUID) (*dto.PostRes, error) {
	post, err := uc.postRepo.GetByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.Status == poststatus.Deleted {
		return nil, communityerrors.ErrPostNotFound()
	}
	if post.Status == poststatus.Hidden && (viewerUserID == nil || *viewerUserID != post.UserID) {
		return nil, communityerrors.ErrPostNotFound()
	}

	var outfit *wardrobe_dto.OutfitContractDTO
	if post.OutfitID != nil {
		if fetched, ferr := uc.outfitContract.GetOutfitByID(ctx, *post.OutfitID); ferr == nil {
			outfit = fetched
		}
	}
	return uc.enrichSingle(ctx, post, viewerUserID, outfit)
}

func (uc *PostUseCase) GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.PostListRes, error) {
	feedType := feedtype.FeedType(strings.ToLower(strings.TrimSpace(query.Type)))
	if feedType == "" || !feedType.IsValid() {
		feedType = feedtype.Explore
	}
	sort := feedsort.FeedSort(strings.ToLower(strings.TrimSpace(query.Sort)))
	if sort == "" || !sort.IsValid() {
		sort = feedsort.Hot
	}

	filter := repositories.PostFilter{
		ViewerUserID:  viewerUserID,
		Type:          feedType,
		Sort:          sort,
		HotWindowDays: uc.cfg.Community.HotWindowDays,
		HotGravity:    uc.cfg.Community.HotGravity,
		LikeWeight:    uc.cfg.Community.LikeWeight,
		CommentWeight: uc.cfg.Community.CommentWeight,
		Page:          query.Page,
		Limit:         query.Limit,
	}

	if query.PostType != "" {
		pt := posttype.PostType(strings.ToLower(strings.TrimSpace(query.PostType)))
		if pt.IsValid() {
			filter.PostType = &pt
		}
	}
	if feedType == feedtype.Following {
		if viewerUserID == nil {
			return nil, apperror.ErrUnauthorized()
		}
		filter.FollowingOf = viewerUserID
	}
	if query.Username != "" {
		authorID, err := uc.resolveUsername(ctx, query.Username)
		if err != nil {
			return nil, err
		}
		if authorID == nil {
			return emptyFeed(query), nil
		}
		filter.AuthorUserID = authorID
	}

	result, err := uc.postRepo.ListFeed(ctx, filter)
	if err != nil {
		return nil, err
	}
	items, err := uc.enrichList(ctx, result.Items, viewerUserID)
	if err != nil {
		return nil, err
	}
	return &dto.PostListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: query.Page, Limit: query.Limit}, result.TotalCount),
	}, nil
}

func (uc *PostUseCase) resolveUsername(ctx context.Context, username string) (*uuid.UUID, error) {
	author, err := usecase_shared.ResolveUserByUsername(ctx, uc.userBrief, username)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, nil
	}
	return &author.ID, nil
}

func (uc *PostUseCase) GetUserPosts(ctx context.Context, username string, viewerUserID *uuid.UUID, query dto.GetUserPostsQueryReq) (*dto.PostListRes, error) {
	author, err := usecase_shared.ResolveUserByUsername(ctx, uc.userBrief, username)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, communityerrors.ErrUserNotFound()
	}
	result, err := uc.postRepo.ListByAuthorPaginated(ctx, author.ID, viewerUserID, query.PaginationQuery)
	if err != nil {
		return nil, err
	}
	items, err := uc.enrichList(ctx, result.Items, viewerUserID)
	if err != nil {
		return nil, err
	}
	return &dto.PostListRes{
		Items:    items,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}

func emptyFeed(query dto.GetFeedQueryReq) *dto.PostListRes {
	return &dto.PostListRes{
		Items:    []*dto.PostRes{},
		Metadata: shared_dto.BuildPaginationMetadata(shared_dto.PaginationQuery{Page: query.Page, Limit: query.Limit}, 0),
	}
}
