package wardrobe

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"

	"github.com/google/uuid"
)

func (uc *WardrobeUseCase) CloneWardrobeItem(ctx context.Context, userID uuid.UUID, id uuid.UUID, quantity int) ([]*dto.WardrobeItemRes, error) {
	if quantity < 1 || quantity > 5 {
		return nil, errorcode.NewBadRequest("Số lượng bản sao nhân bản chỉ được phép từ 1 đến 5 cái.")
	}

	// 1. Kiểm tra giới hạn số lượng trang phục của gói cước
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if int(currentCount)+quantity > subOverview.MaxWardrobeItems {
		return nil, errorcode.NewForbidden(fmt.Sprintf("Hành động nhân bản (%d cái) vượt quá giới hạn của gói hiện tại (Tối đa: %d, Hiện có: %d). Vui lòng nâng cấp Premium!", quantity, subOverview.MaxWardrobeItems, currentCount))
	}

	// 2. Lấy trang phục gốc
	original, err := uc.wardrobeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy trang phục gốc để thực hiện nhân bản.")
	}

	// Ràng buộc bảo mật: Chỉ cho phép nhân bản trang phục của chính mình
	if original.UserID != userID {
		return nil, errorcode.NewForbidden("Bạn không có quyền nhân bản trang phục của người khác.")
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
