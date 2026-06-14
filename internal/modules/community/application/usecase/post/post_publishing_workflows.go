package post

import (
	"context"
	"strings"

	community_dto "smart-wardrobe-be/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/internal/modules/community/application/errors"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

// CreatePost creates a community post and synchronizes all related post items and media.
func (uc *UserPostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input community_dto.CreatePostReq) (*community_dto.PostRes, error) {
	normalizedPostType, resolvedItems, err := uc.preparePostCreate(ctx, userID, input)
	if err != nil {
		return nil, err
	}

	publicID, err := uc.generateUniquePostPublicID(ctx)
	if err != nil {
		return nil, err
	}

	postEntity := &entities.Post{
		UserID:      userID,
		PublicID:    publicID,
		PostType:    normalizedPostType,
		Title:       input.Title,
		Content:     input.Content,
		ContactInfo: input.ContactInfo,
	}

	if err := uc.publishing.uow.Execute(ctx, func(txCtx context.Context) error {
		return uc.persistNewPostAggregate(txCtx, postEntity, normalizedPostType, resolvedItems, input.Media)
	}); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, postEntity.PublicID, &userID)
}

// UpdatePost updates an existing user-owned post and applies item/media diffs transactionally.
func (uc *UserPostUseCase) UpdatePost(ctx context.Context, userID uuid.UUID, postPublicID string, input community_dto.UpdatePostReq) (*community_dto.PostRes, error) {
	postEntity, resolvedItems, currentItems, err := uc.preparePostUpdate(ctx, userID, postPublicID, input)
	if err != nil {
		return nil, err
	}
	postEntity.Title = input.Title
	postEntity.Content = input.Content
	postEntity.ContactInfo = input.ContactInfo

	if err := uc.publishing.uow.Execute(ctx, func(txCtx context.Context) error {
		return uc.persistUpdatedPostAggregate(txCtx, postEntity, resolvedItems, currentItems, input.Media)
	}); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, postEntity.PublicID, &userID)
}

// preparePostCreate validates and resolves all inputs required for post creation.
func (uc *UserPostUseCase) preparePostCreate(ctx context.Context, userID uuid.UUID, input community_dto.CreatePostReq) (posttype.PostType, []resolvedPostItemInput, error) {
	normalizedPostType, err := uc.normalizePostType(input.PostType)
	if err != nil {
		return "", nil, err
	}
	if err := uc.validateCreatePostInput(normalizedPostType, input.Content, input.ContactInfo, input.Items); err != nil {
		return "", nil, err
	}

	itemIDs := uc.collectPostItemIDs(input.Items)
	if err := uc.publishing.wardrobeCtr.VerifyItemsForPost(ctx, userID, itemIDs); err != nil {
		return "", nil, err
	}

	resolvedItems, err := uc.resolvePostItems(ctx, normalizedPostType, input.Items)
	if err != nil {
		return "", nil, err
	}

	if normalizedPostType == posttype.Sale {
		activeTransfers, err := uc.publishing.postItemRepo.GetActiveTransfersByItemIDs(ctx, itemIDs)
		if err != nil {
			return "", nil, err
		}
		if len(activeTransfers) > 0 {
			return "", nil, communityerrors.ErrActiveTransferExists
		}
	}

	return normalizedPostType, resolvedItems, nil
}

// preparePostUpdate loads the target post and resolves the next aggregate state.
func (uc *UserPostUseCase) preparePostUpdate(
	ctx context.Context,
	userID uuid.UUID,
	postPublicID string,
	input community_dto.UpdatePostReq,
) (*entities.Post, []resolvedPostItemInput, []*entities.PostItem, error) {
	normalizedPublicID, err := NormalizePostPublicID(postPublicID)
	if err != nil {
		return nil, nil, nil, err
	}

	postEntity, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return nil, nil, nil, err
	}
	if postEntity == nil || postEntity.UserID != userID {
		return nil, nil, nil, communityerrors.ErrPostNotFound
	}

	if err := uc.validateCreatePostInput(postEntity.PostType, input.Content, input.ContactInfo, input.Items); err != nil {
		return nil, nil, nil, err
	}

	itemIDs := uc.collectPostItemIDs(input.Items)
	if err := uc.publishing.wardrobeCtr.VerifyItemsForPost(ctx, userID, itemIDs); err != nil {
		return nil, nil, nil, err
	}

	resolvedItems, err := uc.resolvePostItems(ctx, postEntity.PostType, input.Items)
	if err != nil {
		return nil, nil, nil, err
	}

	currentItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, postEntity.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	return postEntity, resolvedItems, currentItems, nil
}

// persistNewPostAggregate writes the post, post items, media, and wardrobe side effects.
func (uc *UserPostUseCase) persistNewPostAggregate(
	txCtx context.Context,
	postEntity *entities.Post,
	postType posttype.PostType,
	resolvedItems []resolvedPostItemInput,
	media []community_dto.PostMediaReq,
) error {
	if err := uc.publishing.postRepo.Create(txCtx, postEntity); err != nil {
		return err
	}
	if err := uc.publishing.postRepo.MarkHotnessDirty(txCtx, postEntity.ID); err != nil {
		return err
	}

	if _, err := uc.replacePostItems(txCtx, postEntity.ID, nil, resolvedItems); err != nil {
		return err
	}
	if err := uc.replacePostMedia(txCtx, postEntity.ID, media); err != nil {
		return err
	}
	if postType == posttype.Sale {
		for _, item := range resolvedItems {
			if err := uc.publishing.wardrobeCtr.UpdateItemStatus(txCtx, item.ItemID, wardrobestatus.Selling); err != nil {
				return err
			}
		}
	}

	return uc.syncPostTotalPrice(txCtx, postEntity.ID)
}

// persistUpdatedPostAggregate applies the post aggregate diff and re-syncs affected item states.
func (uc *UserPostUseCase) persistUpdatedPostAggregate(
	txCtx context.Context,
	postEntity *entities.Post,
	resolvedItems []resolvedPostItemInput,
	currentItems []*entities.PostItem,
	media []community_dto.PostMediaReq,
) error {
	return uc.updatePostBaseFields(txCtx, postEntity, media, resolvedItems, currentItems)
}

// updatePostBaseFields updates the post aggregate and re-syncs all side effects.
func (uc *UserPostUseCase) updatePostBaseFields(
	txCtx context.Context,
	postEntity *entities.Post,
	media []community_dto.PostMediaReq,
	resolvedItems []resolvedPostItemInput,
	currentItems []*entities.PostItem,
) error {
	if err := uc.publishing.postRepo.Update(txCtx, postEntity); err != nil {
		return err
	}

	removedWardrobeItems, err := uc.replacePostItems(txCtx, postEntity.ID, currentItems, resolvedItems)
	if err != nil {
		return err
	}
	if err := uc.replacePostMedia(txCtx, postEntity.ID, media); err != nil {
		return err
	}
	if err := uc.syncPostTotalPrice(txCtx, postEntity.ID); err != nil {
		return err
	}
	if err := uc.publishing.postRepo.MarkHotnessDirty(txCtx, postEntity.ID); err != nil {
		return err
	}

	for _, itemID := range removedWardrobeItems {
		if err := SyncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
			return err
		}
	}
	return nil
}

// replacePostItems applies create/update/delete changes for the post's item list.
func (uc *UserPostUseCase) replacePostItems(
	txCtx context.Context,
	postID uuid.UUID,
	currentItems []*entities.PostItem,
	resolvedItems []resolvedPostItemInput,
) ([]uuid.UUID, error) {
	if currentItems == nil {
		for _, item := range resolvedItems {
			if err := uc.publishing.postItemRepo.Create(txCtx, &entities.PostItem{
				PostID:        postID,
				ItemID:        item.ItemID,
				Price:         item.Price,
				ItemCondition: item.ItemCondition,
				Status:        postitemstatus.Available,
			}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	currentByItemID := make(map[uuid.UUID]*entities.PostItem, len(currentItems))
	for _, item := range currentItems {
		currentByItemID[item.ItemID] = item
	}

	nextItemIDs := make(map[uuid.UUID]struct{}, len(resolvedItems))
	for _, item := range resolvedItems {
		nextItemIDs[item.ItemID] = struct{}{}
		existing, ok := currentByItemID[item.ItemID]
		if ok {
			existing.Price = item.Price
			existing.ItemCondition = item.ItemCondition
			existing.Status = postitemstatus.Available
			if err := uc.publishing.postItemRepo.Update(txCtx, existing); err != nil {
				return nil, err
			}
			continue
		}

		if err := uc.publishing.postItemRepo.Create(txCtx, &entities.PostItem{
			PostID:        postID,
			ItemID:        item.ItemID,
			Price:         item.Price,
			ItemCondition: item.ItemCondition,
			Status:        postitemstatus.Available,
		}); err != nil {
			return nil, err
		}
	}

	removedWardrobeItems := make([]uuid.UUID, 0)
	for _, current := range currentItems {
		if _, ok := nextItemIDs[current.ItemID]; ok {
			continue
		}
		removedWardrobeItems = append(removedWardrobeItems, current.ItemID)
		if err := uc.publishing.postItemRepo.Delete(txCtx, current.ID); err != nil {
			return nil, err
		}
	}

	return removedWardrobeItems, nil
}

// replacePostMedia replaces the full media list for the target post.
func (uc *UserPostUseCase) replacePostMedia(txCtx context.Context, postID uuid.UUID, media []community_dto.PostMediaReq) error {
	if err := uc.publishing.postMediaRepo.DeleteByPostID(txCtx, postID); err != nil {
		return err
	}

	mediaItems := make([]*entities.PostMedia, 0, len(media))
	for _, item := range media {
		mediaItems = append(mediaItems, &entities.PostMedia{
			PostID:    postID,
			MediaType: item.MediaType,
			MediaURL:  item.MediaURL,
			PublicID:  item.PublicID,
			SortOrder: item.SortOrder,
		})
	}
	if len(mediaItems) > 0 {
		if err := uc.publishing.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
			return err
		}
	}
	return nil
}

// collectPostItemIDs extracts wardrobe item IDs from the request payload.
func (uc *UserPostUseCase) collectPostItemIDs(items []community_dto.PostItemInputReq) []uuid.UUID {
	itemIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ItemID)
	}
	return itemIDs
}

// GetUploadSignature returns the upload signature used by the post media flow.
func (uc *UserPostUseCase) GetUploadSignature(ctx context.Context) (*community_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: uc.cfg.Cloudinary.PostFolder,
	})
}

// DeletePost soft-deletes a user-owned post and synchronizes wardrobe item statuses.
func (uc *UserPostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, postPublicID string) error {
	normalizedPublicID, err := NormalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	postEntity, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return err
	}
	if postEntity == nil || postEntity.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, postEntity.ID)
	if err != nil {
		return err
	}

	return uc.publishing.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.publishing.postRepo.SoftDelete(txCtx, postEntity.ID); err != nil {
			return err
		}

		for _, itemID := range UniqueItemIDs(postItems) {
			if err := SyncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return nil
	})
}

// RemovePostItems deletes selected post items and updates the parent post aggregate accordingly.
func (uc *UserPostUseCase) RemovePostItems(ctx context.Context, userID uuid.UUID, postPublicID string, postItemIDs []uuid.UUID) error {
	normalizedPublicID, err := NormalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	postEntity, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return err
	}
	if postEntity == nil || postEntity.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	currentItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, postEntity.ID)
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
		if isVisiblePostItem(item) {
			remainingCount++
		}
	}

	return uc.publishing.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.publishing.postItemRepo.DeleteByPostAndIDs(txCtx, postEntity.ID, postItemIDs); err != nil {
			return err
		}
		if err := uc.syncPostTotalPrice(txCtx, postEntity.ID); err != nil {
			return err
		}
		if remainingCount == 0 {
			if err := uc.publishing.postRepo.SoftDelete(txCtx, postEntity.ID); err != nil {
				return err
			}
		}
		for _, itemID := range affectedWardrobeItems {
			if err := SyncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return uc.publishing.postRepo.MarkHotnessDirty(txCtx, postEntity.ID)
	})
}

// validateCreatePostInput validates the required content and sale-specific fields for a post request.
func (uc *UserPostUseCase) validateCreatePostInput(postType posttype.PostType, content string, contactInfo *string, items []community_dto.PostItemInputReq) error {
	if strings.TrimSpace(content) == "" {
		return communityerrors.ErrInvalidPostType
	}
	if postType == posttype.Sale {
		if len(items) == 0 {
			return communityerrors.ErrPostSaleItemsRequired
		}
		if contactInfo == nil || strings.TrimSpace(*contactInfo) == "" {
			return communityerrors.ErrPostContactInfoRequired
		}
	}
	return nil
}

// normalizePostType canonicalizes the requested post type into the domain enum.
func (uc *UserPostUseCase) normalizePostType(raw posttype.PostType) (posttype.PostType, error) {
	switch posttype.PostType(strings.ToUpper(strings.TrimSpace(string(raw)))) {
	case posttype.Outfit:
		return posttype.Outfit, nil
	case posttype.Sale:
		return posttype.Sale, nil
	default:
		return "", communityerrors.ErrInvalidPostType
	}
}
