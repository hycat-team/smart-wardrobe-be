package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *WardrobeUseCase) CopyItemToUser(ctx context.Context, sourceItemID uuid.UUID, targetUserID uuid.UUID) (*dto.WardrobeItemRes, error) {
	sourceItem, err := uc.wardrobeRepo.GetByID(ctx, sourceItemID)
	if err != nil {
		return nil, err
	}
	if sourceItem == nil {
		return nil, apperror.NewNotFound("Không tìm thấy trang phục cần sao chép.")
	}

	cloned := &entities.WardrobeItem{
		UserID:        targetUserID,
		CategoryID:    sourceItem.CategoryID,
		ImageUrl:      sourceItem.ImageUrl,
		ImagePublicID: sourceItem.ImagePublicID,
		Color:         sourceItem.Color,
		Style:         sourceItem.Style,
		Material:      sourceItem.Material,
		Pattern:       sourceItem.Pattern,
		Fit:           sourceItem.Fit,
		Seasonality:   sourceItem.Seasonality,
		Description:   sourceItem.Description,
		Status:        wardrobestatus.InWardrobe,
		ItemType:      itemtype.UserItem,
		Embedding:     sourceItem.Embedding,
	}

	if err := uc.wardrobeRepo.Create(ctx, cloned); err != nil {
		return nil, err
	}

	cloned.Category = sourceItem.Category
	return mapper.MapToWardrobeItemRes(cloned), nil
}

func (uc *WardrobeUseCase) UpdateItemStatus(ctx context.Context, itemID uuid.UUID, status wardrobestatus.WardrobeItemStatus) error {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return apperror.NewNotFound("Không tìm thấy trang phục.")
	}

	item.Status = status
	return uc.wardrobeRepo.Update(ctx, item)
}

func (uc *WardrobeUseCase) VerifyItemsForPost(ctx context.Context, userID uuid.UUID, itemIDs []uuid.UUID) error {
	if len(itemIDs) == 0 {
		return nil
	}

	items, err := uc.wardrobeRepo.GetByIDs(ctx, itemIDs)
	if err != nil {
		return err
	}

	itemMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, item := range items {
		itemMap[item.ID] = item
	}

	for _, itemID := range itemIDs {
		item, ok := itemMap[itemID]
		if !ok {
			return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy trang phục ID %s.", itemID))
		}
		if item.UserID != userID {
			return apperror.NewForbidden(fmt.Sprintf("Trang phục ID %s không thuộc tủ đồ của bạn.", itemID))
		}
		if item.Status == wardrobestatus.Sold {
			return apperror.NewBadRequest(fmt.Sprintf("Trang phục ID %s đã được bán và không thể đăng bài.", itemID))
		}
	}

	return nil
}

