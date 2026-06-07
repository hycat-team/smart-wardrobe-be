package usecase

import (
	"context"
	"math"
	"sort"
	"strings"

	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	community_mapper "smart-wardrobe-be/internal/modules/community/application/mapper"
	shared_repo_dto "smart-wardrobe-be/internal/modules/community/domain/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
)

type scoredPost struct {
	post      *entities.Post
	global    float64
	final     float64
	postItems []*entities.PostItem
	media     []*entities.PostMedia
}

func (uc *UserPostUseCase) GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query community_dto.GetFeedQueryReq) (*community_dto.GetFeedRes, error) {
	feedQuery, err := uc.normalizeFeedQuery(query)
	if err != nil {
		return nil, err
	}

	if feedQuery.Sort == "hot" && viewerUserID != nil {
		return uc.getPersonalizedHotFeed(ctx, *viewerUserID, feedQuery)
	}

	feedResult, err := uc.reader.postRepo.GetFeed(ctx, feedQuery)
	if err != nil {
		return nil, err
	}

	postIDs := make([]uuid.UUID, 0, len(feedResult.Items))
	for _, record := range feedResult.Items {
		postIDs = append(postIDs, record.Post.ID)
	}

	likedMap := make(map[uuid.UUID]bool)
	if viewerUserID != nil {
		likedMap, err = uc.reader.likeRepo.GetLikedPostIDs(ctx, *viewerUserID, postIDs)
		if err != nil {
			return nil, err
		}
	}

	items := make([]*community_dto.PostRes, 0, len(feedResult.Items))
	for _, record := range feedResult.Items {
		items = append(items, community_mapper.MapPost(
			record.Post,
			nil,
			nil,
			likedMap[record.Post.ID],
			record.GlobalHotnessScore,
			record.GlobalHotnessScore,
		))
	}

	return &community_dto.GetFeedRes{
		Items:    items,
		Metadata: feedResult.Metadata,
	}, nil
}

func (uc *UserPostUseCase) getPersonalizedHotFeed(ctx context.Context, viewerUserID uuid.UUID, feedQuery shared_repo_dto.FeedQuery) (*community_dto.GetFeedRes, error) {
	records, err := uc.reader.postRepo.GetHotFeedCandidates(ctx, feedQuery, maxPersonalizedWindow)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &community_dto.GetFeedRes{
			Items: []*community_dto.PostRes{},
			Metadata: shared_persist.BuildPaginationMetadata(shared_dto.PaginationQuery{
				Page:  feedQuery.Page,
				Limit: feedQuery.Limit,
			}, 0),
		}, nil
	}

	user, err := uc.reader.userRepo.GetByID(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.StyleProfile == nil || len(user.StyleProfile.TasteEmbedding) == 0 {
		items := make([]*community_dto.PostRes, 0, len(records))
		for _, record := range records {
			items = append(items, community_mapper.MapPost(record.Post, nil, nil, false, record.GlobalHotnessScore, record.GlobalHotnessScore))
		}
		items, _ = uc.applyLikeStatus(ctx, viewerUserID, items)
		return paginateFeed(items, shared_dto.PaginationQuery{
			Page:  feedQuery.Page,
			Limit: feedQuery.Limit,
		}), nil
	}

	scoredItems := make([]*scoredPost, 0, len(records))
	for _, record := range records {
		post, items, media, err := uc.reader.postRepo.GetDetail(ctx, record.Post.PublicID)
		if err != nil {
			return nil, err
		}
		if post == nil {
			continue
		}

		styleScore := computeStyleScore(user.StyleProfile.TasteEmbedding, items)
		finalScore := (record.GlobalHotnessScore * 0.4) + (styleScore * 0.6)
		scoredItems = append(scoredItems, &scoredPost{
			post:      post,
			global:    record.GlobalHotnessScore,
			final:     finalScore,
			postItems: items,
			media:     media,
		})
	}

	sort.SliceStable(scoredItems, func(i, j int) bool {
		if scoredItems[i].final == scoredItems[j].final {
			return scoredItems[i].post.CreatedAt.After(scoredItems[j].post.CreatedAt)
		}
		return scoredItems[i].final > scoredItems[j].final
	})

	postIDs := make([]uuid.UUID, 0, len(scoredItems))
	for _, item := range scoredItems {
		postIDs = append(postIDs, item.post.ID)
	}
	likedMap, err := uc.reader.likeRepo.GetLikedPostIDs(ctx, viewerUserID, postIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*community_dto.PostRes, 0, len(scoredItems))
	for _, item := range scoredItems {
		items = append(items, community_mapper.MapPost(
			item.post,
			item.postItems,
			item.media,
			likedMap[item.post.ID],
			item.global,
			item.final,
		))
	}

	return paginateFeed(items, shared_dto.PaginationQuery{
		Page:  feedQuery.Page,
		Limit: feedQuery.Limit,
	}), nil
}

func (uc *UserPostUseCase) GetPostDetail(ctx context.Context, postPublicID string, viewerUserID *uuid.UUID) (*community_dto.PostRes, error) {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	post, items, media, err := uc.reader.postRepo.GetDetail(ctx, normalizedPublicID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, communityerrors.ErrPostNotFound
	}

	scoreMap, err := uc.reader.postScoreRepo.GetScoresByPostIDs(ctx, []uuid.UUID{post.ID})
	if err != nil {
		return nil, err
	}

	isLiked := false
	if viewerUserID != nil {
		likedMap, err := uc.reader.likeRepo.GetLikedPostIDs(ctx, *viewerUserID, []uuid.UUID{post.ID})
		if err != nil {
			return nil, err
		}
		isLiked = likedMap[post.ID]
	}

	score := scoreMap[post.ID]
	return community_mapper.MapPost(post, items, media, isLiked, score, score), nil
}

func (uc *UserPostUseCase) GetPostComments(ctx context.Context, postPublicID string) ([]*community_dto.CommentRes, error) {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	post, err := uc.reader.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, communityerrors.ErrPostNotFound
	}

	items, err := uc.reader.commentRepo.GetTopLevelByPostID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	result := make([]*community_dto.CommentRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapCommentRes(item))
	}
	return result, nil
}

func (uc *UserPostUseCase) GetCommentReplies(ctx context.Context, postPublicID string, commentID uuid.UUID) ([]*community_dto.CommentRes, error) {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	post, err := uc.reader.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, communityerrors.ErrPostNotFound
	}

	parentComment, err := uc.reader.commentRepo.GetByIDAndPostID(ctx, commentID, post.ID)
	if err != nil {
		return nil, err
	}
	if parentComment == nil || parentComment.ParentCommentID != nil {
		return nil, communityerrors.ErrCommentReplyTargetInvalid
	}

	items, err := uc.reader.commentRepo.GetRepliesByParentID(ctx, post.ID, commentID)
	if err != nil {
		return nil, err
	}

	result := make([]*community_dto.CommentRes, 0, len(items))
	for _, item := range items {
		result = append(result, mapCommentRes(item))
	}
	return result, nil
}

func (uc *UserPostUseCase) GetPostLikes(ctx context.Context, postPublicID string) ([]*community_dto.PostLikeUserRes, error) {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	post, err := uc.reader.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, communityerrors.ErrPostNotFound
	}

	users, err := uc.reader.likeRepo.GetUsersByPostID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	result := make([]*community_dto.PostLikeUserRes, 0, len(users))
	for _, user := range users {
		result = append(result, mapLikeUserRes(user))
	}
	return result, nil
}

func (uc *UserPostUseCase) applyLikeStatus(ctx context.Context, viewerUserID uuid.UUID, items []*community_dto.PostRes) ([]*community_dto.PostRes, error) {
	postIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		postIDs = append(postIDs, item.ID)
	}
	likedMap, err := uc.reader.likeRepo.GetLikedPostIDs(ctx, viewerUserID, postIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.IsLiked = likedMap[item.ID]
	}
	return items, nil
}

func paginateFeed(items []*community_dto.PostRes, pagination shared_dto.PaginationQuery) *community_dto.GetFeedRes {
	pagination = shared_persist.NormalizePagination(pagination)
	start := shared_persist.Offset(pagination)
	if start >= len(items) {
		return &community_dto.GetFeedRes{
			Items:    []*community_dto.PostRes{},
			Metadata: shared_persist.BuildPaginationMetadata(pagination, int64(len(items))),
		}
	}

	end := start + pagination.Limit
	if end > len(items) {
		end = len(items)
	}

	return &community_dto.GetFeedRes{
		Items:    items[start:end],
		Metadata: shared_persist.BuildPaginationMetadata(pagination, int64(len(items))),
	}
}

func (uc *UserPostUseCase) normalizeFeedQuery(query community_dto.GetFeedQueryReq) (shared_repo_dto.FeedQuery, error) {
	result := shared_repo_dto.FeedQuery{
		Sort:  strings.ToLower(strings.TrimSpace(query.Sort)),
		Page:  query.Page,
		Limit: query.Limit,
	}
	if result.Sort == "" {
		result.Sort = "hot"
	}
	if result.Sort != "hot" && result.Sort != "newest" {
		return shared_repo_dto.FeedQuery{}, communityerrors.ErrInvalidSortCriterion
	}
	if result.Page <= 0 {
		result.Page = 1
	}
	if result.Limit <= 0 {
		result.Limit = defaultFeedLimit
	}
	if query.Username != "" {
		username := strings.TrimSpace(query.Username)
		result.Username = &username
	}
	if query.PostType != "" {
		postType, err := uc.normalizePostType(query.PostType)
		if err != nil {
			return shared_repo_dto.FeedQuery{}, err
		}
		postTypeStr := string(postType)
		result.PostType = &postTypeStr
	}
	return result, nil
}

func computeStyleScore(userVector entities.Vector, items []*entities.PostItem) float64 {
	if len(userVector) == 0 {
		return 0
	}

	best := 0.0
	for _, item := range items {
		if item == nil || item.WardrobeItem == nil || len(item.WardrobeItem.Embedding) == 0 {
			continue
		}

		distance := cosineDistance(userVector, item.WardrobeItem.Embedding)
		if distance < 0 {
			distance = 0
		}
		if distance > 2 {
			distance = 2
		}

		score := 1 - (distance / 2)
		if score > best {
			best = score
		}
	}

	return best
}

func cosineDistance(a, b entities.Vector) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 2
	}

	n := min(len(b), len(a))

	var dot, magA, magB float64
	for i := 0; i < n; i++ {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		magA += av * av
		magB += bv * bv
	}

	if magA == 0 || magB == 0 {
		return 2
	}

	cosineSimilarity := dot / (math.Sqrt(magA) * math.Sqrt(magB))
	return 1 - cosineSimilarity
}
