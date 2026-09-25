package post

import (
	"context"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

// SearchPosts matches published posts whose first ContentPrefixChars characters
// contain the keyword (FR-018), ordered by hot.
func (uc *PostUseCase) SearchPosts(ctx context.Context, query dto.SearchQueryReq, viewerUserID *uuid.UUID) (*dto.PostListRes, error) {
	keyword := strings.TrimSpace(query.Q)
	if keyword == "" {
		return &dto.PostListRes{
			Items:    []*dto.PostRes{},
			Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, 0),
		}, nil
	}

	filter := repositories.SearchPostsFilter{
		Keyword:            keyword,
		ContentPrefixChars: uc.cfg.Community.SearchContentPrefixChars,
		HotWindowDays:      uc.cfg.Community.HotWindowDays,
		HotGravity:         uc.cfg.Community.HotGravity,
		LikeWeight:         uc.cfg.Community.LikeWeight,
		CommentWeight:      uc.cfg.Community.CommentWeight,
		Page:               query.Page,
		Limit:              query.Limit,
	}
	if query.PostType != "" {
		pt := posttype.PostType(strings.ToLower(strings.TrimSpace(query.PostType)))
		if pt.IsValid() {
			filter.PostType = &pt
		}
	}

	result, err := uc.postRepo.Search(ctx, filter)
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
