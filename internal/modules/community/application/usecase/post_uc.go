package usecase

import (
	"context"
	"math"
	"sort"
	"strings"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/application/mapper"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	identity_repos "smart-wardrobe-be/internal/modules/identity/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/itemcondition"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
)

const (
	defaultFeedLimit      = 20
	maxPersonalizedWindow = 100
)

type scoredPost struct {
	post      *entities.Post
	global    float64
	final     float64
	isLiked   bool
	postItems []*entities.PostItem
	media     []*entities.PostMedia
	comments  []*entities.Comment
}

type PostUseCase struct {
	cfg           *config.Config
	postRepo      repositories.IPostRepository
	postScoreRepo repositories.IPostScoreRepository
	postItemRepo  repositories.IPostItemRepository
	postMediaRepo repositories.IPostMediaRepository
	commentRepo   repositories.ICommentRepository
	likeRepo      repositories.ILikeRepository
	userRepo      identity_repos.IUserRepository
	wardrobeCtr   wardrobe_contract.IWardrobeContract
	mediaService  media.IMediaService
	uow           shared_repos.IUnitOfWork
}

func NewPostUseCase(
	cfg *config.Config,
	postRepo repositories.IPostRepository,
	postScoreRepo repositories.IPostScoreRepository,
	postItemRepo repositories.IPostItemRepository,
	postMediaRepo repositories.IPostMediaRepository,
	commentRepo repositories.ICommentRepository,
	likeRepo repositories.ILikeRepository,
	userRepo identity_repos.IUserRepository,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	mediaService media.IMediaService,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IPostUseCase {
	return &PostUseCase{
		cfg:           cfg,
		postRepo:      postRepo,
		postScoreRepo: postScoreRepo,
		postItemRepo:  postItemRepo,
		postMediaRepo: postMediaRepo,
		commentRepo:   commentRepo,
		likeRepo:      likeRepo,
		userRepo:      userRepo,
		wardrobeCtr:   wardrobeCtr,
		mediaService:  mediaService,
		uow:           uow,
	}
}

func (uc *PostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error) {
	normalizedPostType, err := uc.normalizePostType(input.PostType)
	if err != nil {
		return nil, err
	}
	if err := uc.validateCreatePostInput(normalizedPostType, input); err != nil {
		return nil, err
	}

	if err := uc.wardrobeCtr.VerifyItemsForPost(ctx, userID, input.ItemIDs); err != nil {
		return nil, err
	}

	if normalizedPostType == posttype.Sale {
		for _, itemID := range input.ItemIDs {
			hasActiveTransfer, err := uc.postItemRepo.HasActiveTransfer(ctx, itemID, nil)
			if err != nil {
				return nil, err
			}
			if hasActiveTransfer {
				return nil, apperror.NewBadRequest("Trang phục này đang có giao dịch chờ xử lý, không thể đăng thêm listing mới.")
			}
		}
	}

	post := &entities.Post{
		UserID:   userID,
		PostType: normalizedPostType,
		Title:    input.Title,
		Content:  input.Content,
	}
	if input.ContactInfo != nil {
		post.ContactInfo = input.ContactInfo
	}
	if input.TotalPrice != nil {
		post.TotalPrice = *input.TotalPrice
	}

	createPost := func(txCtx context.Context) error {
		if err := uc.postRepo.Create(txCtx, post); err != nil {
			return err
		}

		postItems := make([]*entities.PostItem, 0, len(input.ItemIDs))
		for _, itemID := range input.ItemIDs {
			postItems = append(postItems, &entities.PostItem{
				PostID:        post.ID,
				ItemID:        itemID,
				Price:         post.TotalPrice,
				ItemCondition: itemcondition.Standard,
				Status:        postitemstatus.Available,
			})
		}
		for _, item := range postItems {
			if err := uc.postItemRepo.Create(txCtx, item); err != nil {
				return err
			}
		}

		mediaItems := make([]*entities.PostMedia, 0, len(input.Media))
		for _, item := range input.Media {
			mediaItems = append(mediaItems, &entities.PostMedia{
				PostID:    post.ID,
				MediaType: item.MediaType,
				MediaURL:  item.MediaURL,
				PublicID:  item.PublicID,
				SortOrder: item.SortOrder,
			})
		}
		if len(mediaItems) > 0 {
			if err := uc.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
				return err
			}
		}

		if normalizedPostType == posttype.Sale {
			for _, itemID := range input.ItemIDs {
				if err := uc.wardrobeCtr.UpdateItemStatus(txCtx, itemID, wardrobestatus.Selling); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if err := uc.uow.Execute(ctx, createPost); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, post.ID, &userID)
}

func (uc *PostUseCase) GetFeed(ctx context.Context, viewerUserID *uuid.UUID, query dto.GetFeedQueryReq) (*dto.GetFeedRes, error) {
	feedQuery, err := uc.normalizeFeedQuery(query)
	if err != nil {
		return nil, err
	}

	if feedQuery.Sort == "hot" && viewerUserID != nil {
		return uc.getPersonalizedHotFeed(ctx, *viewerUserID, feedQuery)
	}

	feedResult, err := uc.postRepo.GetFeed(ctx, feedQuery)
	if err != nil {
		return nil, err
	}

	postIDs := make([]uuid.UUID, 0, len(feedResult.Items))
	for _, record := range feedResult.Items {
		postIDs = append(postIDs, record.Post.ID)
	}

	likedMap := make(map[uuid.UUID]bool)
	if viewerUserID != nil {
		likedMap, err = uc.likeRepo.GetLikedPostIDs(ctx, *viewerUserID, postIDs)
		if err != nil {
			return nil, err
		}
	}

	items := make([]*dto.PostRes, 0, len(feedResult.Items))
	for _, record := range feedResult.Items {
		items = append(items, mapper.MapPost(
			record.Post,
			nil,
			nil,
			nil,
			likedMap[record.Post.ID],
			record.GlobalHotnessScore,
			record.GlobalHotnessScore,
		))
	}

	return &dto.GetFeedRes{
		Items:    items,
		Metadata: feedResult.Metadata,
	}, nil
}

func (uc *PostUseCase) getPersonalizedHotFeed(ctx context.Context, viewerUserID uuid.UUID, feedQuery repositories.FeedQuery) (*dto.GetFeedRes, error) {
	records, err := uc.postRepo.GetHotFeedCandidates(ctx, feedQuery, maxPersonalizedWindow)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &dto.GetFeedRes{
			Items: []*dto.PostRes{},
			Metadata: shared_persist.BuildPaginationMetadata(shared_dto.PaginationQuery{
				Page:  feedQuery.Page,
				Limit: feedQuery.Limit,
			}, 0),
		}, nil
	}

	user, err := uc.userRepo.GetByID(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.StyleProfile == nil || len(user.StyleProfile.TasteEmbedding) == 0 {
		items := make([]*dto.PostRes, 0, len(records))
		for _, record := range records {
			items = append(items, mapper.MapPost(record.Post, nil, nil, nil, false, record.GlobalHotnessScore, record.GlobalHotnessScore))
		}
		items, _ = uc.applyLikeStatus(ctx, viewerUserID, items)
		return paginateFeed(items, shared_dto.PaginationQuery{
			Page:  feedQuery.Page,
			Limit: feedQuery.Limit,
		}), nil
	}

	scoredItems := make([]*scoredPost, 0, len(records))
	for _, record := range records {
		post, items, media, err := uc.postRepo.GetDetail(ctx, record.Post.ID)
		if err != nil {
			return nil, err
		}
		if post == nil {
			continue
		}
		comments, err := uc.commentRepo.GetByPostID(ctx, record.Post.ID)
		if err != nil {
			return nil, err
		}

		styleScore := computeStyleScore(user.StyleProfile.TasteEmbedding, items)
		finalScore := (record.GlobalHotnessScore * 0.4) + (styleScore * 0.6)
		scoredItems = append(scoredItems, &scoredPost{
			post:      post,
			global:    record.GlobalHotnessScore,
			final:     finalScore,
			postItems: items,
			media:     media,
			comments:  comments,
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
	likedMap, err := uc.likeRepo.GetLikedPostIDs(ctx, viewerUserID, postIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.PostRes, 0, len(scoredItems))
	for _, item := range scoredItems {
		items = append(items, mapper.MapPost(
			item.post,
			item.postItems,
			item.media,
			item.comments,
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

func (uc *PostUseCase) applyLikeStatus(ctx context.Context, viewerUserID uuid.UUID, items []*dto.PostRes) ([]*dto.PostRes, error) {
	postIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		postIDs = append(postIDs, item.ID)
	}
	likedMap, err := uc.likeRepo.GetLikedPostIDs(ctx, viewerUserID, postIDs)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.IsLiked = likedMap[item.ID]
	}
	return items, nil
}

func paginateFeed(items []*dto.PostRes, pagination shared_dto.PaginationQuery) *dto.GetFeedRes {
	pagination = shared_persist.NormalizePagination(pagination)
	start := shared_persist.Offset(pagination)
	if start >= len(items) {
		return &dto.GetFeedRes{
			Items:    []*dto.PostRes{},
			Metadata: shared_persist.BuildPaginationMetadata(pagination, int64(len(items))),
		}
	}

	end := start + pagination.Limit
	if end > len(items) {
		end = len(items)
	}

	return &dto.GetFeedRes{
		Items:    items[start:end],
		Metadata: shared_persist.BuildPaginationMetadata(pagination, int64(len(items))),
	}
}

func (uc *PostUseCase) GetPostDetail(ctx context.Context, postID uuid.UUID, viewerUserID *uuid.UUID) (*dto.PostRes, error) {
	post, items, media, err := uc.postRepo.GetDetail(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperror.NewNotFound("Không tìm thấy bài đăng.")
	}

	comments, err := uc.commentRepo.GetByPostID(ctx, postID)
	if err != nil {
		return nil, err
	}

	scoreMap, err := uc.postScoreRepo.GetScoresByPostIDs(ctx, []uuid.UUID{postID})
	if err != nil {
		return nil, err
	}

	isLiked := false
	if viewerUserID != nil {
		likedMap, err := uc.likeRepo.GetLikedPostIDs(ctx, *viewerUserID, []uuid.UUID{postID})
		if err != nil {
			return nil, err
		}
		isLiked = likedMap[postID]
	}

	score := scoreMap[postID]
	return mapper.MapPost(post, items, media, comments, isLiked, score, score), nil
}

func (uc *PostUseCase) GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error) {
	folder := uc.cfg.Cloudinary.PostFolder

	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: folder,
	})
}

func (uc *PostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return apperror.NewNotFound("Không tìm thấy bài đăng.")
	}

	postItems, err := uc.postItemRepo.GetByPostID(ctx, postID)
	if err != nil {
		return err
	}

	deletePost := func(txCtx context.Context) error {
		if err := uc.postRepo.Delete(txCtx, postID); err != nil {
			return err
		}

		affectedItemIDs := uniqueItemIDs(postItems)
		for _, itemID := range affectedItemIDs {
			if err := uc.syncWardrobeStatusByItem(txCtx, itemID); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.uow.Execute(ctx, deletePost)
}

func (uc *PostUseCase) RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error {
	post, err := uc.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return apperror.NewNotFound("Không tìm thấy bài đăng.")
	}

	currentItems, err := uc.postItemRepo.GetByPostID(ctx, postID)
	if err != nil {
		return err
	}

	targetIDs := make(map[uuid.UUID]struct{}, len(postItemIDs))
	for _, id := range postItemIDs {
		targetIDs[id] = struct{}{}
	}

	affectedWardrobeItems := make([]uuid.UUID, 0, len(postItemIDs))
	remainingCount := 0
	for _, item := range currentItems {
		if _, ok := targetIDs[item.ID]; ok {
			affectedWardrobeItems = append(affectedWardrobeItems, item.ItemID)
			continue
		}
		remainingCount++
	}

	removePostItems := func(txCtx context.Context) error {
		if err := uc.postItemRepo.DeleteByPostAndIDs(txCtx, postID, postItemIDs); err != nil {
			return err
		}

		if remainingCount == 0 {
			if err := uc.postRepo.Delete(txCtx, postID); err != nil {
				return err
			}
		}

		for _, itemID := range affectedWardrobeItems {
			if err := uc.syncWardrobeStatusByItem(txCtx, itemID); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.uow.Execute(ctx, removePostItems)
}

func (uc *PostUseCase) validateCreatePostInput(postType posttype.PostType, input dto.CreatePostReq) error {
	if postType == posttype.Sale {
		if len(input.ItemIDs) == 0 {
			return apperror.NewBadRequest("Sell post bắt buộc phải có ít nhất một item.")
		}
		if input.ContactInfo == nil || strings.TrimSpace(*input.ContactInfo) == "" {
			return apperror.NewBadRequest("Sell post bắt buộc phải có contactInfo.")
		}
	}
	return nil
}

func (uc *PostUseCase) normalizePostType(raw string) (posttype.PostType, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(posttype.Outfit):
		return posttype.Outfit, nil
	case string(posttype.Sale):
		return posttype.Sale, nil
	default:
		return "", apperror.NewBadRequest("Loại bài đăng không hợp lệ.")
	}
}

func (uc *PostUseCase) normalizeFeedQuery(query dto.GetFeedQueryReq) (repositories.FeedQuery, error) {
	result := repositories.FeedQuery{
		Sort:  strings.ToLower(strings.TrimSpace(query.Sort)),
		Page:  query.Page,
		Limit: query.Limit,
	}
	if result.Sort == "" {
		result.Sort = "hot"
	}
	if result.Sort != "hot" && result.Sort != "newest" {
		return repositories.FeedQuery{}, apperror.NewBadRequest("Giá trị sort không hợp lệ.")
	}
	if result.Page <= 0 {
		result.Page = 1
	}
	if result.Limit <= 0 {
		result.Limit = defaultFeedLimit
	}
	if query.UserID != "" {
		parsed, err := uuid.Parse(query.UserID)
		if err != nil {
			return repositories.FeedQuery{}, apperror.NewBadRequest("userId không hợp lệ.")
		}
		result.UserID = &parsed
	}
	if query.PostType != "" {
		postType, err := uc.normalizePostType(query.PostType)
		if err != nil {
			return repositories.FeedQuery{}, err
		}
		postTypeStr := string(postType)
		result.PostType = &postTypeStr
	}
	return result, nil
}

func (uc *PostUseCase) syncWardrobeStatusByItem(ctx context.Context, itemID uuid.UUID) error {
	postItems, err := uc.postItemRepo.GetByItemID(ctx, itemID)
	if err != nil {
		return err
	}

	nextStatus := wardrobestatus.InWardrobe
	for _, item := range postItems {
		if item.TransferState == transferstate.Accepted || item.Status == postitemstatus.Sold {
			nextStatus = wardrobestatus.Sold
			break
		}
		if item.Status == postitemstatus.Available || item.TransferState == transferstate.Pending {
			nextStatus = wardrobestatus.Selling
		}
	}

	return uc.wardrobeCtr.UpdateItemStatus(ctx, itemID, nextStatus)
}

func uniqueItemIDs(items []*entities.PostItem) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(items))
	result := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.ItemID]; ok {
			continue
		}
		seen[item.ItemID] = struct{}{}
		result = append(result, item.ItemID)
	}
	return result
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

	n := len(a)
	if len(b) < n {
		n = len(b)
	}

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

var _ uc_interfaces.IPostUseCase = (*PostUseCase)(nil)
