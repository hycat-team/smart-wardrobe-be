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

func (uc *WardrobeUseCase) InitClosetFromCatalog(ctx context.Context, userID uuid.UUID, catalogItemIDs []uuid.UUID) ([]*dto.WardrobeItemRes, error) {
	if len(catalogItemIDs) == 0 {
		return nil, apperror.NewBadRequest("Danh sách trang phục mẫu không được để trống.")
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

	if int(currentCount)+len(catalogItemIDs) > subOverview.MaxWardrobeItems {
		return nil, apperror.NewForbidden(fmt.Sprintf("Số lượng trang phục khởi tạo vượt quá giới hạn tủ đồ của gói dịch vụ hiện tại (Tủ đồ: %d/%d, yêu cầu thêm: %d).", currentCount, subOverview.MaxWardrobeItems, len(catalogItemIDs)))
	}

	// 2. Fetch system template catalog items from Database
	templates, err := uc.wardrobeRepo.GetByIDs(ctx, catalogItemIDs)
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, apperror.NewNotFound("Không tìm thấy trang phục mẫu phù hợp.")
	}

	newItems := make([]*entities.WardrobeItem, len(templates))
	for i, original := range templates {
		newItems[i] = &entities.WardrobeItem{
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
			ItemType:      itemtype.UserItem, // Once copied to the user's wardrobe, it becomes a personal UserItem
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, newItems)
	if err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i := 0; i < len(newItems); i++ {
		newItems[i].Category = templates[i].Category
		resList[i] = mapper.MapToWardrobeItemRes(newItems[i])
		resList[i].IsLocked = false
	}

	return resList, nil
}
