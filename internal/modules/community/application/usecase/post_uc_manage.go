package usecase

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

func (uc *UserPostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input community_dto.CreatePostReq) (*community_dto.PostRes, error) {
	normalizedPostType, err := uc.normalizePostType(input.PostType)
	if err != nil {
		return nil, err
	}
	if err := uc.validateCreatePostInput(normalizedPostType, input.Content, input.ContactInfo, input.Items); err != nil {
		return nil, err
	}

	itemIDs := make([]uuid.UUID, 0, len(input.Items))
	for _, item := range input.Items {
		itemIDs = append(itemIDs, item.ItemID)
	}
	if err := uc.publishing.wardrobeCtr.VerifyItemsForPost(ctx, userID, itemIDs); err != nil {
		return nil, err
	}

	resolvedItems, err := uc.resolvePostItems(ctx, normalizedPostType, input.Items)
	if err != nil {
		return nil, err
	}

	if normalizedPostType == posttype.Sale {
		activeTransfers, err := uc.publishing.postItemRepo.GetActiveTransfersByItemIDs(ctx, itemIDs)
		if err != nil {
			return nil, err
		}
		if len(activeTransfers) > 0 {
			return nil, communityerrors.ErrActiveTransferExists
		}
	}

	publicID, err := uc.generateUniquePostPublicID(ctx)
	if err != nil {
		return nil, err
	}

	post := &entities.Post{
		UserID:      userID,
		PublicID:    publicID,
		PostType:    normalizedPostType,
		Title:       input.Title,
		Content:     input.Content,
		ContactInfo: input.ContactInfo,
	}

	createPost := func(txCtx context.Context) error {
		if err := uc.publishing.postRepo.Create(txCtx, post); err != nil {
			return err
		}
		if err := uc.publishing.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
			return err
		}

		postItems := make([]*entities.PostItem, 0, len(resolvedItems))
		for _, item := range resolvedItems {
			postItems = append(postItems, &entities.PostItem{
				PostID:        post.ID,
				ItemID:        item.ItemID,
				Price:         item.Price,
				ItemCondition: item.ItemCondition,
				Status:        postitemstatus.Available,
			})
		}
		for _, item := range postItems {
			if err := uc.publishing.postItemRepo.Create(txCtx, item); err != nil {
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
			if err := uc.publishing.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
				return err
			}
		}

		if normalizedPostType == posttype.Sale {
			for _, item := range resolvedItems {
				if err := uc.publishing.wardrobeCtr.UpdateItemStatus(txCtx, item.ItemID, wardrobestatus.Selling); err != nil {
					return err
				}
			}
		}

		return uc.syncPostTotalPrice(txCtx, post.ID)
	}

	if err := uc.publishing.uow.Execute(ctx, createPost); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, post.PublicID, &userID)
}

func (uc *UserPostUseCase) UpdatePost(ctx context.Context, userID uuid.UUID, postPublicID string, input community_dto.UpdatePostReq) (*community_dto.PostRes, error) {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return nil, err
	}

	post, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return nil, err
	}
	if post == nil || post.UserID != userID {
		return nil, communityerrors.ErrPostNotFound
	}

	if err := uc.validateCreatePostInput(post.PostType, input.Content, input.ContactInfo, input.Items); err != nil {
		return nil, err
	}

	itemIDs := make([]uuid.UUID, 0, len(input.Items))
	for _, item := range input.Items {
		itemIDs = append(itemIDs, item.ItemID)
	}
	if err := uc.publishing.wardrobeCtr.VerifyItemsForPost(ctx, userID, itemIDs); err != nil {
		return nil, err
	}

	resolvedItems, err := uc.resolvePostItems(ctx, post.PostType, input.Items)
	if err != nil {
		return nil, err
	}

	currentItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, post.ID)
	if err != nil {
		return nil, err
	}

	updatePost := func(txCtx context.Context) error {
		post.Title = input.Title
		post.Content = input.Content
		post.ContactInfo = input.ContactInfo
		if err := uc.publishing.postRepo.Update(txCtx, post); err != nil {
			return err
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
					return err
				}
				continue
			}

			if err := uc.publishing.postItemRepo.Create(txCtx, &entities.PostItem{
				PostID:        post.ID,
				ItemID:        item.ItemID,
				Price:         item.Price,
				ItemCondition: item.ItemCondition,
				Status:        postitemstatus.Available,
			}); err != nil {
				return err
			}
		}

		removedWardrobeItems := make([]uuid.UUID, 0)
		for _, current := range currentItems {
			if _, ok := nextItemIDs[current.ItemID]; ok {
				continue
			}
			removedWardrobeItems = append(removedWardrobeItems, current.ItemID)
			if err := uc.publishing.postItemRepo.Delete(txCtx, current.ID); err != nil {
				return err
			}
		}

		if err := uc.publishing.postMediaRepo.DeleteByPostID(txCtx, post.ID); err != nil {
			return err
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
			if err := uc.publishing.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
				return err
			}
		}

		if err := uc.syncPostTotalPrice(txCtx, post.ID); err != nil {
			return err
		}
		if err := uc.publishing.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range removedWardrobeItems {
			if err := syncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
				return err
			}
		}

		return nil
	}

	if err := uc.publishing.uow.Execute(ctx, updatePost); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, post.PublicID, &userID)
}

func (uc *UserPostUseCase) GetUploadSignature(ctx context.Context) (*community_dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: uc.cfg.Cloudinary.PostFolder,
	})
}

func (uc *UserPostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, postPublicID string) error {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	post, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, post.ID)
	if err != nil {
		return err
	}

	deletePost := func(txCtx context.Context) error {
		if err := uc.publishing.postRepo.SoftDelete(txCtx, post.ID); err != nil {
			return err
		}

		for _, itemID := range uniqueItemIDs(postItems) {
			if err := syncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.publishing.uow.Execute(ctx, deletePost)
}

func (uc *UserPostUseCase) RemovePostItems(ctx context.Context, userID uuid.UUID, postPublicID string, postItemIDs []uuid.UUID) error {
	normalizedPublicID, err := normalizePostPublicID(postPublicID)
	if err != nil {
		return err
	}

	post, err := uc.publishing.postRepo.GetByPublicID(ctx, normalizedPublicID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	currentItems, err := uc.publishing.postItemRepo.GetByPostID(ctx, post.ID)
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

	removePostItems := func(txCtx context.Context) error {
		if err := uc.publishing.postItemRepo.DeleteByPostAndIDs(txCtx, post.ID, postItemIDs); err != nil {
			return err
		}

		if err := uc.syncPostTotalPrice(txCtx, post.ID); err != nil {
			return err
		}

		if remainingCount == 0 {
			if err := uc.publishing.postRepo.SoftDelete(txCtx, post.ID); err != nil {
				return err
			}
		}

		for _, itemID := range affectedWardrobeItems {
			if err := syncWardrobeStatusByItem(txCtx, uc.publishing.postItemRepo, uc.publishing.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return uc.publishing.postRepo.MarkHotnessDirty(txCtx, post.ID)
	}

	return uc.publishing.uow.Execute(ctx, removePostItems)
}

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
