package item

import (
	"context"
	"encoding/json"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/shared/application/sse"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

type BrandFashionEventUseCase struct {
	brandItemRepo     repositories.IBrandItemRepository
	realtimePublisher sse.IRealtimePublisher
	logger            logger.Interface
}

func NewBrandFashionEventUseCase(
	brandItemRepo repositories.IBrandItemRepository,
	realtimePublisher sse.IRealtimePublisher,
	logger logger.Interface,
) uc_interfaces.IBrandFashionEventUseCase {
	return &BrandFashionEventUseCase{
		brandItemRepo:     brandItemRepo,
		realtimePublisher: realtimePublisher,
		logger:            logger,
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

	// Idempotency: only update DB if the terminal result changes persisted state.
	// Clearing TaskID marks the analysis session as finished so the client stops
	// subscribing to a stream that will never emit again.
	if item.Status != targetStatus || item.TaskID != nil {
		item.Status = targetStatus
		item.TaskID = nil
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
			_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
		}
	}

	return nil
}

// HandleFashionEvent dispatch theo status payload. Trước đây nằm ở
// presentation worker — chuyển vào usecase để worker thin.
func (uc *BrandFashionEventUseCase) HandleFashionEvent(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	switch payload.Status {
	case analyzetaskstatus.Failed:
		return uc.HandleFashionFailed(ctx, payload)
	case analyzetaskstatus.Processing:
		return uc.HandleFashionProcessing(ctx, payload)
	default:
		return uc.HandleFashionAnalyzed(ctx, payload)
	}
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

	if item.Status != branditemstatus.Failed || item.TaskID != nil {
		item.Status = branditemstatus.Failed
		item.TaskID = nil
		if err := uc.brandItemRepo.Update(ctx, item); err != nil {
			uc.logger.Error("Failed to update brand item status to Failed",
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
			_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
		}
	}

	return nil
}

// HandleFashionProcessing emits a realtime "processing" event for a brand item
// without changing persisted state (FR-004/FR-013). Idempotent by design.
func (uc *BrandFashionEventUseCase) HandleFashionProcessing(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload) error {
	item, err := uc.brandItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch brand item for processing event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Brand item not found for processing event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
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
	uc.emitSSE(ctx, payload, brief)
	return nil
}

// emitSSE marshals and best-effort publishes an SSE progress event for a task.
func (uc *BrandFashionEventUseCase) emitSSE(ctx context.Context, payload fashion_dto.FashionAnalyzedEventPayload, brief *dto.BrandItemBriefRes) {
	if payload.TaskID == "" {
		return
	}
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
		return
	}
	_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
}

// HandleFashionProcessingExhausted moves a brand item to the terminal `failed`
// state once automatic retries are exhausted, so it never gets stuck in draft
// (FR-034). The reason is recorded on the fashion item by the producer.
func (uc *BrandFashionEventUseCase) HandleFashionProcessingExhausted(ctx context.Context, payload fashion_dto.FashionProcessingExhaustedEventPayload) error {
	item, err := uc.brandItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch brand item for processing exhausted event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Brand item not found for processing exhausted event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}
	if item.Status == branditemstatus.Failed {
		return nil
	}
	item.Status = branditemstatus.Failed
	item.UpdatedAt = time.Now().UTC()
	if err := uc.brandItemRepo.Update(ctx, item); err != nil {
		uc.logger.Error("Failed to update brand item status to Failed for processing exhausted",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (uc *BrandFashionEventUseCase) HandleFashionItemCreated(ctx context.Context, payload fashion_dto.FashionItemCreatedEventPayload) error {
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
