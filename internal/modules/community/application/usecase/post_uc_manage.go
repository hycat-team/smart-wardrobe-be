package usecase

import (
	"context"
	"strings"

	"smart-wardrobe-be/internal/modules/community/application/dto"
	"smart-wardrobe-be/internal/modules/community/application/errors"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemcondition"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *UserPostUseCase) CreatePost(ctx context.Context, userID uuid.UUID, input dto.CreatePostReq) (*dto.PostRes, error) {
	normalizedPostType, err := uc.normalizePostType(input.PostType)
	if err != nil {
		return nil, err
	}
	if err := uc.validateCreatePostInput(normalizedPostType, input); err != nil {
		return nil, err
	}

	if err := uc.writer.wardrobeCtr.VerifyItemsForPost(ctx, userID, input.ItemIDs); err != nil {
		return nil, err
	}

	if normalizedPostType == posttype.Sale {
		for _, itemID := range input.ItemIDs {
			hasActiveTransfer, err := uc.writer.postItemRepo.HasActiveTransfer(ctx, itemID, nil)
			if err != nil {
				return nil, err
			}
			if hasActiveTransfer {
				return nil, communityerrors.ErrActiveTransferExists
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
		if err := uc.writer.postRepo.Create(txCtx, post); err != nil {
			return err
		}
		if err := uc.writer.postRepo.MarkHotnessDirty(txCtx, post.ID); err != nil {
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
			if err := uc.writer.postItemRepo.Create(txCtx, item); err != nil {
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
			if err := uc.writer.postMediaRepo.BulkCreate(txCtx, mediaItems); err != nil {
				return err
			}
		}

		if normalizedPostType == posttype.Sale {
			for _, itemID := range input.ItemIDs {
				if err := uc.writer.wardrobeCtr.UpdateItemStatus(txCtx, itemID, wardrobestatus.Selling); err != nil {
					return err
				}
			}
		}

		return nil
	}

	if err := uc.writer.uow.Execute(ctx, createPost); err != nil {
		return nil, err
	}

	return uc.GetPostDetail(ctx, post.ID, &userID)
}

func (uc *UserPostUseCase) GetUploadSignature(ctx context.Context) (*dto.UploadSignatureResult, error) {
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder: uc.cfg.Cloudinary.PostFolder,
	})
}

func (uc *UserPostUseCase) DeletePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	post, err := uc.writer.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	postItems, err := uc.writer.postItemRepo.GetByPostID(ctx, postID)
	if err != nil {
		return err
	}

	deletePost := func(txCtx context.Context) error {
		if err := uc.writer.postRepo.Delete(txCtx, postID); err != nil {
			return err
		}

		affectedItemIDs := uniqueItemIDs(postItems)
		for _, itemID := range affectedItemIDs {
			if err := syncWardrobeStatusByItem(txCtx, uc.writer.postItemRepo, uc.writer.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.writer.uow.Execute(ctx, deletePost)
}

func (uc *UserPostUseCase) RemovePostItems(ctx context.Context, userID uuid.UUID, postID uuid.UUID, postItemIDs []uuid.UUID) error {
	post, err := uc.writer.postRepo.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || post.UserID != userID {
		return communityerrors.ErrPostNotFound
	}

	currentItems, err := uc.writer.postItemRepo.GetByPostID(ctx, postID)
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
		if err := uc.writer.postItemRepo.DeleteByPostAndIDs(txCtx, postID, postItemIDs); err != nil {
			return err
		}

		if remainingCount == 0 {
			if err := uc.writer.postRepo.Delete(txCtx, postID); err != nil {
				return err
			}
		}

		for _, itemID := range affectedWardrobeItems {
			if err := syncWardrobeStatusByItem(txCtx, uc.writer.postItemRepo, uc.writer.wardrobeCtr, itemID); err != nil {
				return err
			}
		}
		return nil
	}

	return uc.writer.uow.Execute(ctx, removePostItems)
}

func (uc *UserPostUseCase) validateCreatePostInput(postType posttype.PostType, input dto.CreatePostReq) error {
	if postType == posttype.Sale {
		if len(input.ItemIDs) == 0 {
			return communityerrors.ErrPostSaleItemsRequired
		}
		if input.ContactInfo == nil || strings.TrimSpace(*input.ContactInfo) == "" {
			return communityerrors.ErrPostContactInfoRequired
		}
	}
	return nil
}

func (uc *UserPostUseCase) normalizePostType(raw string) (posttype.PostType, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(posttype.Outfit):
		return posttype.Outfit, nil
	case string(posttype.Sale):
		return posttype.Sale, nil
	default:
		return "", communityerrors.ErrInvalidPostType
	}
}

func filterVisiblePostItems(items []*entities.PostItem) []*entities.PostItem {
	result := make([]*entities.PostItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.Status == postitemstatus.Hidden {
			continue
		}
		result = append(result, item)
	}
	return result
}
