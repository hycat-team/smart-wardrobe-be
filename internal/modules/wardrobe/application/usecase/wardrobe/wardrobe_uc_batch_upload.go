package wardrobe

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (uc *WardrobeWorkerUseCase) BatchUploadWardrobeItems(ctx context.Context, userID uuid.UUID, currentRole roleslug.RoleSlug, input dto.BatchUploadWardrobeItemsReq) ([]*dto.WardrobeItemRes, error) {
	if len(input.Items) == 0 {
		return nil, wardrobeerrors.ErrUploadImagesEmpty
	}

	itemType := itemtype.SystemCatalogItem
	if currentRole == roleslug.Member {
		itemType = itemtype.UserItem
		subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
		if err != nil {
			return nil, err
		}

		currentCount, err := uc.wardrobeRepo.CountByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if int(currentCount)+len(input.Items) > subOverview.MaxWardrobeItems {
			return nil, wardrobeerrors.ErrWardrobeLimitExceededForUpload(int(currentCount), subOverview.MaxWardrobeItems, len(input.Items))
		}
	}

	newItems := make([]*entities.WardrobeItem, len(input.Items))
	for i, itemReq := range input.Items {
		newItems[i] = &entities.WardrobeItem{
			UserID:        userID,
			CategoryID:    itemReq.CategoryID,
			ImageUrl:      itemReq.ImageUrl,
			ImagePublicID: itemReq.ImagePublicID,
			Status:        wardrobestatus.Processing,
			ItemType:      itemType,
		}
	}

	if err := uc.wardrobeRepo.BulkCreate(ctx, newItems); err != nil {
		return nil, err
	}

	resList := make([]*dto.WardrobeItemRes, len(newItems))
	for i, item := range newItems {
		job := dto.WardrobeBatchUploadJobDTO{
			ItemID:        item.ID,
			UserID:        userID,
			CategoryID:    item.CategoryID,
			ImageUrl:      item.ImageUrl,
			ImagePublicID: item.ImagePublicID,
		}

		err := uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeBatchUpload, job)
		if err != nil {
			uc.logger.Error("[WardrobeBatchUploadUseCase] Event publishing failed", zap.Error(err))
			item.Status = wardrobestatus.Failed
			_ = uc.wardrobeRepo.Update(ctx, item)
		}

		resList[i] = mapper.MapToWardrobeItemRes(item)
		resList[i].IsLocked = false
	}

	return resList, nil
}

func (uc *WardrobeWorkerUseCase) handleJobFailure(ctx context.Context, job dto.WardrobeBatchUploadJobDTO, err error) {
	if isTransientError(err) && job.RetryCount < 3 {
		job.RetryCount++

		baseDelay := 60 * time.Second
		switch job.RetryCount {
		case 2:
			baseDelay = 300 * time.Second
		case 3:
			baseDelay = 900 * time.Second
		}

		jitter := time.Duration(1+rand.Intn(10)) * time.Second
		delay := baseDelay + jitter

		uc.logger.Warn(fmt.Sprintf("[WardrobeBatchUploadWorker] Transient error encountered. Scheduling retry %d/3 in %v", job.RetryCount, delay),
			zap.String("item_id", job.ItemID.String()),
			zap.Error(err),
		)

		time.AfterFunc(delay, func() {
			publishCtx := context.Background()
			if republishErr := uc.eventPublisher.Publish(publishCtx, eventconstants.TopicWardrobeBatchUpload, job); republishErr != nil {
				uc.logger.Error("[WardrobeBatchUploadWorker] Failed to republish retry job", zap.Error(republishErr))
			}
		})
		return
	}

	uc.logger.Error(fmt.Sprintf("[WardrobeBatchUploadWorker] Fatal error or max retries exceeded. Marking item as Failed. RetryCount: %d", job.RetryCount),
		zap.String("item_id", job.ItemID.String()),
		zap.Error(err),
	)
	uc.markJobFailed(ctx, job.ItemID)
}

func (uc *WardrobeWorkerUseCase) ProcessBackgroundBatchUploadJob(ctx context.Context, job dto.WardrobeBatchUploadJobDTO) error {
	categories, err := uc.categoryRepo.GetAll(ctx)
	if err != nil {
		uc.handleJobFailure(ctx, job, fmt.Errorf("category lookup failed: %w", err))
		return nil
	}

	aiCatRefs := make([]shared_dto.AICategoryRef, len(categories))
	categoryMap := make(map[string]uuid.UUID)
	categoryNameMap := make(map[uuid.UUID]string)
	var otherCategoryID uuid.UUID

	for i, cat := range categories {
		aiCatRefs[i] = shared_dto.AICategoryRef{
			Name: cat.Name,
			Slug: cat.Slug,
		}
		categoryMap[cat.Slug] = cat.ID
		categoryNameMap[cat.ID] = cat.Name
		if cat.Slug == "other" {
			otherCategoryID = cat.ID
		}
	}

	aiMeta, err := uc.aiService.AnalyzeFashionImage(ctx, job.ImageUrl, aiCatRefs)
	if err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	var detectedCategoryID *uuid.UUID
	detectedCategoryName := "Khác"

	if id, exists := categoryMap[aiMeta.CategorySlug]; exists {
		detectedCategoryID = &id
		if name, existsName := categoryNameMap[id]; existsName {
			detectedCategoryName = name
		}
	} else if otherCategoryID != uuid.Nil {
		detectedCategoryID = &otherCategoryID
		if name, existsName := categoryNameMap[otherCategoryID]; existsName {
			detectedCategoryName = name
		}
	}

	richTextContext := fmt.Sprintf(
		"Danh mục trang phục: %s, Thuộc tính màu sắc: %s, Định hình phong cách thiết kế: %s, Chất liệu: %s, Họa tiết: %s, Kiểu dáng: %s, Mùa phù hợp: %s. Mô tả chi tiết: %s",
		detectedCategoryName,
		aiMeta.Color,
		aiMeta.Style,
		aiMeta.Material,
		aiMeta.Pattern,
		aiMeta.Fit,
		aiMeta.Seasonality,
		aiMeta.Description,
	)

	embeddings, err := uc.aiService.GenerateEmbeddings(ctx, []string{richTextContext})
	if err != nil || len(embeddings) == 0 {
		if err == nil {
			err = fmt.Errorf("no embeddings returned")
		}
		uc.handleJobFailure(ctx, job, err)
		return nil
	}
	embedding := embeddings[0]

	item, err := uc.wardrobeRepo.GetByID(ctx, job.ItemID)
	if err != nil || item == nil {
		if err == nil {
			err = fmt.Errorf("item not found in database")
		}
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	item.CategoryID = detectedCategoryID
	item.Color = &aiMeta.Color
	item.Style = &aiMeta.Style
	item.Material = &aiMeta.Material
	item.Pattern = &aiMeta.Pattern
	item.Fit = &aiMeta.Fit
	item.Seasonality = &aiMeta.Seasonality
	item.Description = &aiMeta.Description
	item.Embedding = entities.Vector(embedding)
	item.Status = wardrobestatus.InWardrobe

	if err := uc.wardrobeRepo.Update(ctx, item); err != nil {
		uc.handleJobFailure(ctx, job, err)
		return nil
	}

	payload := dto.WardrobeEventPayload{
		ItemID: item.ID,
		UserID: item.UserID,
		Action: eventconstants.ActionCreated,
	}
	_ = uc.eventPublisher.Publish(ctx, eventconstants.TopicWardrobeCreated, payload)

	return nil
}

func (uc *WardrobeWorkerUseCase) markJobFailed(ctx context.Context, itemID uuid.UUID) {
	item, err := uc.wardrobeRepo.GetByID(ctx, itemID)
	if err == nil && item != nil {
		item.Status = wardrobestatus.Failed
		_ = uc.wardrobeRepo.Update(ctx, item)
	}
}
