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
	"go.uber.org/zap"
)

func (uc *WardrobeUseCase) BatchCropWardrobeItems(ctx context.Context, userID uuid.UUID, input dto.BatchCropWardrobeItemsReq) ([]*dto.WardrobeItemRes, error) {
	if len(input.Items) == 0 {
		return nil, errorcode.NewBadRequest("Danh sách ảnh cắt không được để trống.")
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

	if int(currentCount)+len(input.Items) > subOverview.MaxWardrobeItems {
		return nil, errorcode.NewForbidden(fmt.Sprintf("Hành động số hóa hàng loạt (%d trang phục) vượt quá giới hạn của gói hiện tại (Tối đa: %d, Hiện có: %d). Vui lòng nâng cấp Premium!", len(input.Items), subOverview.MaxWardrobeItems, currentCount))
	}

	// 2. Tạo nhanh các trang phục mẫu trong DB ở trạng thái Processing
	newItems := make([]*entities.WardrobeItem, len(input.Items))
	for i, itemReq := range input.Items {
		newItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			CategoryID:    itemReq.CategoryID,
			ImageUrl:      itemReq.ImageUrl,
			ImagePublicID: itemReq.ImagePublicID,
			Status:        wardrobestatus.Processing, // Trạng thái xử lý AI ngầm
			ItemType:      itemtype.UserItem,
		}
	}

	err = uc.wardrobeRepo.BulkCreate(ctx, newItems)
	if err != nil {
		return nil, err
	}

	// 3. Đẩy event qua bộ gửi sự kiện trừu tượng (Clean Architecture)
	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i, item := range newItems {
		job := dto.BatchCropJobDTO{
			ItemID:        item.ID,
			UserID:        userID,
			CategoryID:    item.CategoryID,
			ImageUrl:      item.ImageUrl,
			ImagePublicID: item.ImagePublicID,
		}

		err = uc.eventPublisher.Publish(ctx, "batch_crop_jobs", job)
		if err != nil {
			uc.logger.Error("[BatchCropUseCase] Event publishing failed", zap.Error(err))
			item.Status = wardrobestatus.Failed
			_ = uc.wardrobeRepo.Update(ctx, item)
		}

		resList[i] = mapper.MapToWardrobeItemRes(item)
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeUseCase) ProcessBackgroundCropJob(ctx context.Context, job dto.BatchCropJobDTO) error {
	// 1. Fetch category details
	category, err := uc.categoryRepo.GetByID(ctx, job.CategoryID)
	if err != nil || category == nil {
		uc.markJobFailed(ctx, job.ItemID)
		return fmt.Errorf("category lookup failed")
	}

	// 2. Vision AI analysis (sử dụng Rate Limiter của AIService)
	aiMeta, err := uc.aiService.AnalyzeFashionImage(ctx, job.ImageUrl)
	if err != nil {
		uc.markJobFailed(ctx, job.ItemID)
		return err
	}

	richTextContext := fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		category.Name,
		aiMeta.Color,
		aiMeta.Style,
		aiMeta.Material,
		aiMeta.Pattern,
		aiMeta.Fit,
		aiMeta.Seasonality,
		aiMeta.Description,
	)

	// 3. Generate Embedding Vector
	embeddings, err := uc.aiService.GenerateEmbeddings(ctx, []string{richTextContext})
	if err != nil || len(embeddings) == 0 {
		uc.markJobFailed(ctx, job.ItemID)
		return fmt.Errorf("embeddings generation failed")
	}
	embedding := embeddings[0]

	// 4. Update item inside DB with dynamic properties
	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil || item == nil {
		return fmt.Errorf("item not found in database")
	}

	item.Color = &aiMeta.Color
	item.Style = &aiMeta.Style
	item.Material = &aiMeta.Material
	item.Pattern = &aiMeta.Pattern
	item.Fit = &aiMeta.Fit
	item.Seasonality = &aiMeta.Seasonality
	item.Description = &aiMeta.Description
	item.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe

	err = uc.wardrobeRepo.Update(ctx, item)
	if err != nil {
		return err
	}

	return nil
}

func (uc *WardrobeUseCase) markJobFailed(ctx context.Context, itemID uuid.UUID) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err == nil && item != nil {
		item.Status = wardrobestatus.Failed
		_ = uc.wardrobeRepo.Update(ctx, item)
	}
}
