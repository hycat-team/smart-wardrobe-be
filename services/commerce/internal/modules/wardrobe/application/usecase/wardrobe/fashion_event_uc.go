package wardrobe

import (
	"context"
	"encoding/json"

	fashionDto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/application/sse"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

type WardrobeFashionEventUseCase struct {
	wardrobeItemRepo  repositories.IWardrobeItemRepository
	realtimePublisher sse.IRealtimePublisher
	eventPublisher    event.IEventPublisher
	uow               shared_repos.IUnitOfWork
	logger            logger.Interface
}

func NewWardrobeFashionEventUseCase(
	wardrobeItemRepo repositories.IWardrobeItemRepository,
	realtimePublisher sse.IRealtimePublisher,
	eventPublisher event.IEventPublisher,
	uow shared_repos.IUnitOfWork,
	logger logger.Interface,
) uc_interfaces.IWardrobeFashionEventUseCase {
	return &WardrobeFashionEventUseCase{
		wardrobeItemRepo:  wardrobeItemRepo,
		realtimePublisher: realtimePublisher,
		eventPublisher:    eventPublisher,
		uow:               uow,
		logger:            logger,
	}
}

// HandleFashionEvent dispatch theo status payload. Trước đây nằm ở
// presentation worker — chuyển vào usecase để worker thin
// và dispatch được test độc lập.
func (uc *WardrobeFashionEventUseCase) HandleFashionEvent(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
	switch payload.Status {
	case analyzetaskstatus.Failed:
		return uc.HandleFashionFailed(ctx, payload)
	case analyzetaskstatus.Processing:
		return uc.HandleFashionProcessing(ctx, payload)
	default:
		return uc.HandleFashionAnalyzed(ctx, payload)
	}
}

func (uc *WardrobeFashionEventUseCase) HandleFashionAnalyzed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for analyzed event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for analyzed event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}

	targetStatus := wardrobestatus.InWardrobe
	if payload.Status == analyzetaskstatus.NeedsReview {
		targetStatus = wardrobestatus.NeedsReview
	}

	// Idempotency: only update DB and emit created domain event if item status changed (atomic via outbox)
	if item.Status != targetStatus {
		if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
			item.Status = targetStatus
			if err := uc.wardrobeItemRepo.Update(txCtx, item); err != nil {
				uc.logger.Error("Failed to update wardrobe item status",
					zap.String("itemId", item.ID.String()),
					zap.Error(err),
				)
				return err
			}
			if targetStatus == wardrobestatus.InWardrobe {
				wardrobeCreatedPayload := dto.WardrobeEventPayload{
					ItemID: item.ID,
					UserID: item.UserID,
					Action: eventconstants.ActionCreated,
				}
				return uc.eventPublisher.Publish(txCtx, eventconstants.TopicWardrobeCreated, wardrobeCreatedPayload)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	brief := &dto.WardrobeItemBriefRes{
		ID:          item.ID,
		FashionItem: payload.FashionBrief,
		Price:       item.Price,
		Status:      item.Status,
		LastUsedAt:  item.LastUsedAt,
		TaskID:      item.TaskID,
	}

	if payload.TaskID != "" {
		ssePayload := fashionDto.SSEEventPayload{
			ItemID: payload.ItemID,
			Status: payload.Status,
			Total:  payload.Total,
			Index:  payload.Index,
			Data:   brief,
			Error:  payload.Error,
		}
		bytes, err := json.Marshal(ssePayload)
		if err != nil {
			uc.logger.Error("Failed to marshal SSE payload", zap.Error(err))
		} else {
			_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
		}
	}

	return nil
}

func (uc *WardrobeFashionEventUseCase) HandleFashionFailed(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for failed event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for failed event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}

	if item.Status != wardrobestatus.Failed {
		item.Status = wardrobestatus.Failed
		if err := uc.wardrobeItemRepo.Update(ctx, item); err != nil {
			uc.logger.Error("Failed to update wardrobe item status to Failed",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			return err
		}
	}

	brief := &dto.WardrobeItemBriefRes{
		ID:          item.ID,
		FashionItem: payload.FashionBrief,
		Price:       item.Price,
		Status:      item.Status,
		LastUsedAt:  item.LastUsedAt,
		TaskID:      item.TaskID,
	}

	if payload.TaskID != "" {
		ssePayload := fashionDto.SSEEventPayload{
			ItemID: payload.ItemID,
			Status: analyzetaskstatus.Failed,
			Total:  payload.Total,
			Index:  payload.Index,
			Data:   brief,
			Error:  payload.Error,
		}
		bytes, err := json.Marshal(ssePayload)
		if err != nil {
			uc.logger.Error("Failed to marshal SSE payload", zap.Error(err))
		} else {
			_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
		}
	}

	return nil
}

// HandleFashionProcessing emits a realtime "processing" event for an item
// without changing persisted state (FR-004/FR-013). The event is idempotent:
// repeated deliveries only re-emit the SSE event.
func (uc *WardrobeFashionEventUseCase) HandleFashionProcessing(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for processing event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for processing event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}

	brief := &dto.WardrobeItemBriefRes{
		ID:          item.ID,
		FashionItem: payload.FashionBrief,
		Price:       item.Price,
		Status:      item.Status,
		LastUsedAt:  item.LastUsedAt,
		TaskID:      item.TaskID,
	}
	uc.emitSSE(ctx, payload, brief)
	return nil
}

// emitSSE marshals and best-effort publishes an SSE progress event for a task.
func (uc *WardrobeFashionEventUseCase) emitSSE(ctx context.Context, payload fashionDto.FashionAnalyzedEventPayload, brief *dto.WardrobeItemBriefRes) {
	if payload.TaskID == "" {
		return
	}
	ssePayload := fashionDto.SSEEventPayload{
		ItemID: payload.ItemID,
		Status: payload.Status,
		Total:  payload.Total,
		Index:  payload.Index,
		Data:   brief,
		Error:  payload.Error,
	}
	bytes, err := json.Marshal(ssePayload)
	if err != nil {
		uc.logger.Error("Failed to marshal SSE payload", zap.Error(err))
		return
	}
	_ = uc.realtimePublisher.Publish(ctx, payload.TaskID, bytes)
}

func (uc *WardrobeFashionEventUseCase) HandleFashionItemCreated(ctx context.Context, payload fashionDto.FashionItemCreatedEventPayload) error {
	if payload.ItemType != "" && payload.ItemType != "wardrobe" {
		return nil
	}
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for fashion item created event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for fashion item created event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}
	if item.FashionItemID != nil && *item.FashionItemID == payload.FashionItemID {
		return nil
	}
	item.FashionItemID = &payload.FashionItemID
	if err := uc.wardrobeItemRepo.Update(ctx, item); err != nil {
		uc.logger.Error("Failed to set fashion item id on wardrobe item",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (uc *WardrobeFashionEventUseCase) HandleFashionProcessingExhausted(ctx context.Context, payload fashionDto.FashionProcessingExhaustedEventPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for processing exhausted event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for processing exhausted event",
			zap.String("itemId", payload.ItemID.String()),
		)
		return nil
	}
	if item.Status == wardrobestatus.Failed {
		return nil
	}
	item.Status = wardrobestatus.Failed
	if err := uc.wardrobeItemRepo.Update(ctx, item); err != nil {
		uc.logger.Error("Failed to update wardrobe item status to Failed for processing exhausted",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (uc *WardrobeFashionEventUseCase) HandleWardrobeDeleted(ctx context.Context, payload dto.WardrobeEventPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.ItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for deleted event",
			zap.String("itemId", payload.ItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		return nil
	}
	// Idempotent: if already deleted (soft delete flag or not found after delete), return nil
	// Check IsDeleted if available, otherwise try Delete and ignore not found
	if item.IsDeleted {
		return nil
	}
	if err := uc.wardrobeItemRepo.Delete(ctx, item.ID); err != nil {
		uc.logger.Error("Failed to delete wardrobe item for cleanup",
			zap.String("itemId", item.ID.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}
