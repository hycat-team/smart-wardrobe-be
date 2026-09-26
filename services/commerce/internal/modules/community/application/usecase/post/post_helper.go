package post

import (
	"context"
	"strings"
	"unicode/utf8"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	wardrobe_dto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func buildPostMedia(postID uuid.UUID, reqs []dto.PostMediaReq) []*entities.PostMedia {
	if len(reqs) == 0 {
		return nil
	}
	out := make([]*entities.PostMedia, 0, len(reqs))
	for _, m := range reqs {
		out = append(out, &entities.PostMedia{
			ID:        uuid.New(),
			PostID:    postID,
			MediaType: mediatype.MediaType(m.MediaType),
			MediaURL:  m.MediaURL,
			PublicID:  m.PublicID,
			SortOrder: *m.SortOrder,
		})
	}
	return out
}

func (uc *PostUseCase) validatePostInput(pt posttype.PostType, title *string, content string, media []dto.PostMediaReq) error {
	if !pt.IsValid() {
		return communityerrors.ErrInvalidPostType()
	}
	limits := uc.cfg.Community
	if len(media) > limits.MaxMediaPerPost {
		return communityerrors.ErrMediaLimitExceeded(limits.MaxMediaPerPost)
	}
	if title != nil && utf8.RuneCountInString(*title) > limits.MaxTitleLength {
		return communityerrors.ErrTitleTooLong(limits.MaxTitleLength)
	}
	if utf8.RuneCountInString(content) > limits.MaxContentLength {
		return communityerrors.ErrContentTooLong(limits.MaxContentLength)
	}
	if strings.TrimSpace(content) == "" && len(media) == 0 {
		return communityerrors.ErrEmptyPostBody()
	}
	for _, m := range media {
		if !mediatype.MediaType(m.MediaType).IsValid() {
			return communityerrors.ErrInvalidMedia()
		}
	}
	return validateMediaOrder(media)
}

// validateMediaOrder requires every media item to carry a sort order and
// forbids duplicates. Any int16 value is accepted (no range restriction).
func validateMediaOrder(media []dto.PostMediaReq) error {
	seen := make(map[int16]struct{}, len(media))
	for _, m := range media {
		if m.SortOrder == nil {
			return communityerrors.ErrInvalidMediaOrder()
		}
		if _, dup := seen[*m.SortOrder]; dup {
			return communityerrors.ErrInvalidMediaOrder()
		}
		seen[*m.SortOrder] = struct{}{}
	}
	return nil
}

func (uc *PostUseCase) validateOutfitOwnership(ctx context.Context, userID uuid.UUID, outfitID *uuid.UUID) (*wardrobe_dto.OutfitContractDTO, error) {
	if outfitID == nil {
		return nil, communityerrors.ErrOutfitInvalid()
	}
	outfit, err := uc.outfitContract.GetOutfitByID(ctx, *outfitID)
	if err != nil {
		return nil, err
	}
	if outfit == nil || outfit.UserID != userID {
		return nil, communityerrors.ErrOutfitInvalid()
	}
	return outfit, nil
}

func (uc *PostUseCase) checkRateLimit(ctx context.Context, action string, userID uuid.UUID, limit int) error {
	if uc.rateLimiter == nil || limit <= 0 {
		return nil
	}
	allowed, err := uc.rateLimiter.Allow(ctx, "community:"+action+":"+userID.String(), limit, 3600)
	if err != nil {
		return communityerrors.ErrRateLimited()
	}
	if !allowed {
		return communityerrors.ErrRateLimited()
	}
	return nil
}

// enrichSingle maps one post with author, outfit and viewer flags.
func (uc *PostUseCase) enrichSingle(ctx context.Context, post *entities.Post, viewerID *uuid.UUID, outfit *wardrobe_dto.OutfitContractDTO) (*dto.PostRes, error) {
	author := uc.fetchAuthor(ctx, post.UserID)
	isLiked := false
	if viewerID != nil {
		liked, err := uc.likeRepo.Exists(ctx, post.ID, *viewerID)
		if err != nil {
			return nil, err
		}
		isLiked = liked
	}
	isFollowing := uc.isFollowing(ctx, viewerID, post.UserID)
	return mapper.MapToPostRes(post, author, outfit, isLiked, isFollowing), nil
}

// enrichList batch-enriches posts to avoid N+1 lookups.
func (uc *PostUseCase) enrichList(ctx context.Context, posts []*entities.Post, viewerID *uuid.UUID) ([]*dto.PostRes, error) {
	if len(posts) == 0 {
		return []*dto.PostRes{}, nil
	}

	userIDs := make([]uuid.UUID, 0, len(posts))
	outfitIDs := make([]uuid.UUID, 0, len(posts))
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		userIDs = append(userIDs, p.UserID)
		postIDs = append(postIDs, p.ID)
		if p.OutfitID != nil {
			outfitIDs = append(outfitIDs, *p.OutfitID)
		}
	}

	authors := uc.fetchAuthors(ctx, userIDs)
	outfits := uc.fetchOutfits(ctx, outfitIDs)

	liked := map[uuid.UUID]bool{}
	if viewerID != nil {
		var err error
		liked, err = uc.likeRepo.GetLikedPostIDs(ctx, *viewerID, postIDs)
		if err != nil {
			return nil, err
		}
	}
	following := uc.followeeSet(ctx, viewerID)

	result := make([]*dto.PostRes, 0, len(posts))
	for _, p := range posts {
		var outfit *wardrobe_dto.OutfitContractDTO
		if p.OutfitID != nil {
			outfit = outfits[*p.OutfitID]
		}
		isFollowing := viewerID != nil && *viewerID != p.UserID && following[p.UserID]
		result = append(result, mapper.MapToPostRes(p, authors[p.UserID], outfit, liked[p.ID], isFollowing))
	}
	return result, nil
}

func (uc *PostUseCase) fetchAuthor(ctx context.Context, userID uuid.UUID) *identity_dto.UserBriefContractDTO {
	authors := uc.fetchAuthors(ctx, []uuid.UUID{userID})
	return authors[userID]
}

func (uc *PostUseCase) fetchAuthors(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]*identity_dto.UserBriefContractDTO {
	result := make(map[uuid.UUID]*identity_dto.UserBriefContractDTO, len(userIDs))
	if len(userIDs) == 0 || uc.userBrief == nil {
		return result
	}
	briefs, err := uc.userBrief.GetByIDs(ctx, dedupeUUIDs(userIDs))
	if err != nil {
		uc.logger.Warn("failed to enrich post authors", zap.Error(err))
		return result
	}
	for _, b := range briefs {
		result[b.ID] = b
	}
	return result
}

func (uc *PostUseCase) fetchOutfits(ctx context.Context, outfitIDs []uuid.UUID) map[uuid.UUID]*wardrobe_dto.OutfitContractDTO {
	result := make(map[uuid.UUID]*wardrobe_dto.OutfitContractDTO, len(outfitIDs))
	if len(outfitIDs) == 0 || uc.outfitContract == nil {
		return result
	}
	outfits, err := uc.outfitContract.GetOutfitsByIDs(ctx, dedupeUUIDs(outfitIDs))
	if err != nil {
		uc.logger.Warn("failed to enrich post outfits", zap.Error(err))
		return result
	}
	for _, o := range outfits {
		result[o.ID] = o
	}
	return result
}

func (uc *PostUseCase) followeeSet(ctx context.Context, viewerID *uuid.UUID) map[uuid.UUID]bool {
	result := map[uuid.UUID]bool{}
	if viewerID == nil || uc.followRepo == nil {
		return result
	}
	ids, err := uc.followRepo.GetFolloweeIDs(ctx, *viewerID)
	if err != nil {
		return result
	}
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func (uc *PostUseCase) isFollowing(ctx context.Context, viewerID *uuid.UUID, authorID uuid.UUID) bool {
	if viewerID == nil || *viewerID == authorID {
		return false
	}
	exists, err := uc.followRepo.Exists(ctx, *viewerID, authorID)
	if err != nil {
		return false
	}
	return exists
}

func dedupeUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
