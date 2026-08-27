package wardrobe

import (
	"context"
	"encoding/json"

	fashionDto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/analyzetaskstatus"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/sse"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"go.uber.org/zap"
)

type WardrobeFashionEventUseCase struct {
	wardrobeItemRepo repositories.IWardrobeItemRepository
	sseManager       sse.ISSEManager
	eventPublisher   event.IEventPublisher
	uow              shared_repos.IUnitOfWork
	logger           logger.Interface
}

func NewWardrobeFashionEventUseCase(
	wardrobeItemRepo repositories.IWardrobeItemRepository,
	sseManager sse.ISSEManager,
	eventPublisher event.IEventPublisher,
	uow shared_repos.IUnitOfWork,
	logger logger.Interface,
) uc_interfaces.IWardrobeFashionEventUseCase {
	return &WardrobeFashionEventUseCase{
		wardrobeItemRepo: wardrobeItemRepo,
		sseManager:       sseManager,
		eventPublisher:   eventPublisher,
		uow:              uow,
		logger:           logger,
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
		FashionItem: mapFashionBriefToWardrobe(payload.FashionBrief),
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
			uc.sseManager.Publish(payload.TaskID, bytes)
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
		FashionItem: mapFashionBriefToWardrobe(payload.FashionBrief),
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
			uc.sseManager.Publish(payload.TaskID, bytes)
		}
	}

	return nil
}

func (uc *WardrobeFashionEventUseCase) HandleFashionItemCreated(ctx context.Context, payload fashionDto.FashionItemCreatedPayload) error {
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

func (uc *WardrobeFashionEventUseCase) HandleFashionProcessingExhausted(ctx context.Context, payload fashionDto.FashionProcessingExhaustedPayload) error {
	item, err := uc.wardrobeItemRepo.GetByID(ctx, payload.WardrobeItemID)
	if err != nil {
		uc.logger.Error("Failed to fetch wardrobe item for processing exhausted event",
			zap.String("itemId", payload.WardrobeItemID.String()),
			zap.Error(err),
		)
		return err
	}
	if item == nil {
		uc.logger.Warn("Wardrobe item not found for processing exhausted event",
			zap.String("itemId", payload.WardrobeItemID.String()),
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

func mapFashionBriefToWardrobe(fb *fashionDto.FashionItemBriefRes) *dto.FashionItemBriefRes {
	if fb == nil {
		return nil
	}
	var cat *dto.CategoryBriefRes
	if fb.Category != nil {
		cat = &dto.CategoryBriefRes{
			ID:   fb.Category.ID,
			Name: fb.Category.Name,
			Slug: fb.Category.Slug,
		}
	}
	return &dto.FashionItemBriefRes{
		ID:       fb.ID,
		ImageUrl: fb.ImageUrl,
		Color:    fb.Color,
		ColorHex: fb.ColorHex,
		Style:    fb.Style,
		Category: cat,
	}
}
