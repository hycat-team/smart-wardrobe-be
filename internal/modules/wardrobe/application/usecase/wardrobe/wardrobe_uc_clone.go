package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

func (uc *WardrobeUseCase) CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error) {
	if quantity < 1 || quantity > 5 {
		return nil, apperror.NewBadRequest("Bạn chỉ có thể tạo từ 1 đến 5 bản sao của trang phục này.")
	}

	// 1. Check wardrobe item limit of the subscription plan
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if int(currentCount)+quantity > subOverview.MaxWardrobeItems {
		return nil, apperror.NewForbidden(fmt.Sprintf("Số lượng bản sao vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, subOverview.MaxWardrobeItems, quantity))
	}

	// 2. Retrieve the original wardrobe item
	original, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, apperror.NewNotFound("Không tìm thấy trang phục gốc để sao chép.")
	}

	// Security constraint: Only allow cloning one's own wardrobe items
	if original.UserID != userID {
		return nil, apperror.NewForbidden("Bạn không thể sao chép trang phục của người dùng khác.")
	}

	if original.Status == wardrobestatus.Sold {
		return nil, apperror.NewBadRequest("Không thể sao chép trang phục đã được bán.")
	}

	clonedItems := make([]*entities.WardrobeItem, quantity)
	for i := range quantity {
		clonedItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			CategoryID:    original.CategoryID,
			ImageUrl:      original.ImageUrl,
			ImagePublicID: original.ImagePublicID,
			Color:         original.Color,
			Style:         original.Style,
			Material:      original.Material,
			Pattern:       original.Pattern,
			Fit:           original.Fit,
			Seasonality:   original.Seasonality,
			Description:   original.Description,
			Embedding:     original.Embedding,
			Status:        wardrobestatus.InWardrobe,
			ItemType:      itemtype.UserItem,
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, clonedItems)
	if err != nil {
		return nil, err
	}

	for _, cloned := range clonedItems {
		payload := dto.WardrobeEventPayload{
			ItemID: cloned.ID,
			UserID: cloned.UserID,
			Action: "created",
		}
		_ = uc.eventPublisher.Publish(ctx, "wardrobe.event.created", payload)
	}

	resList := make([]*dto.WardrobeItemRes, quantity)
	for i := range quantity {
		clonedItems[i].Category = original.Category
		resList[i] = mapper.MapToWardrobeItemRes(clonedItems[i])
		resList[i].IsLocked = false
	}

	return resList, nil
}
