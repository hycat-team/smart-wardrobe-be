package item

import (
	"context"
	"encoding/json"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/application/sse"

	"go.uber.org/zap"
)

type BrandFashionEventUseCase struct {
	brandItemRepo repositories.IBrandItemRepository
	sseManager    sse.ISSEManager
	logger        logger.Interface
}

func NewBrandFashionEventUseCase(
	brandItemRepo repositories.IBrandItemRepository,
	sseManager sse.ISSEManager,
	logger logger.Interface,
) uc_interfaces.IBrandFashionEventUseCase {
	return &BrandFashionEventUseCase{
		brandItemRepo: brandItemRepo,
		sseManager:    sseManager,
		logger:        logger,
	}
}

func (uc *BrandFashionEventUseCase) HandleFashionAnalyzed(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	item, err := uc.brandItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch brand item for analyzed event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Brand item not found for analyzed event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}

	targetStatus := branditemstatus.Active
	if payload.Status == analyzetaskstatus.NeedsReview {
		targetStatus = branditemstatus.Draft
	}

	// Idempotency: only update DB if item status changed
	if item.Status != targetStatus {
		item.Status = targetStatus
		if err := uc.brandItemRepo.Update(ctx, item); err != nil {
			uc.logger.Error("Failed to update brand item status",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			return err
		}
	}

	brief := &dto.BrandItemBriefRes{
		ID:          item.ID,
		ProductCode: item.ProductCode,
		Name:        item.Name,
		Price:       item.Price,
		ItemType:    item.ItemType,
		Status:      item.Status,
		FashionItem: payload.FashionBrief,
	}

	if payload.TaskID != "" {
		ssePayload := fashion_dto.SSEEventPayload{
			ItemID: payload.ItemID,
			Status: payload.Status,
			Total:  payload.Total,
			Index:  payload.Index,
			Data:   brief,
			Error:  payload.Error,
		}
		bytes, err := json.Marshal(ssePayload)
		if err != nil {
			uc.logger.Error("Failed to marshal SSE payload for brand item", zap.Error(err))
		} else {
			uc.sseManager.Publish(payload.TaskID, bytes)
		}
	}

	return nil
}

// HandleFashionEvent dispatch theo status payload. Trước đây nằm ở
// presentation worker — chuyển vào usecase để worker thin.
func (uc *BrandFashionEventUseCase) HandleFashionEvent(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	if payload.Status == analyzetaskstatus.Failed {
		return uc.HandleFashionFailed(ctx, payload)
	}
	return uc.HandleFashionAnalyzed(ctx, payload)
}

func (uc *BrandFashionEventUseCase) HandleFashionFailed(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	item, err := uc.brandItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch brand item for failed event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Brand item not found for failed event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}

	if item.Status != branditemstatus.Draft {
		item.Status = branditemstatus.Draft
		if err := uc.brandItemRepo.Update(ctx, item); err != nil {
			uc.logger.Error("Failed to update brand item status to Draft",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			return err
		}
	}

	brief := &dto.BrandItemBriefRes{
		ID:          item.ID,
		ProductCode: item.ProductCode,
		Name:        item.Name,
		Price:       item.Price,
		ItemType:    item.ItemType,
		Status:      item.Status,
		FashionItem: payload.FashionBrief,
	}

	if payload.TaskID != "" {
		ssePayload := fashion_dto.SSEEventPayload{
			ItemID: payload.ItemID,
			Status: analyzetaskstatus.Failed,
			Total:  payload.Total,
			Index:  payload.Index,
			Data:   brief,
			Error:  payload.Error,
		}
		bytes, err := json.Marshal(ssePayload)
		if err != nil {
			uc.logger.Error("Failed to marshal SSE payload for brand item", zap.Error(err))
		} else {
			uc.sseManager.Publish(payload.TaskID, bytes)
		}
	}

	return nil
}

func (uc *BrandFashionEventUseCase) HandleFashionItemCreated(ctx context.Context, payload fashion_dto.FashionItemCreatedPayload) error {
	if payload.ItemType != "" && payload.ItemType != "brand" {
		return nil
	}
	item, err := uc.brandItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch brand item for fashion item created event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Brand item not found for fashion item created event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}
	if item.FashionItemID != nil && *item.FashionItemID == payload.FashionItemID {
		return nil
	}
	item.FashionItemID = &payload.FashionItemID
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		uc.logger.Error("Failed to set fashion item id on brand item",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
